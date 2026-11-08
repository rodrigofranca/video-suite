"""Who is speaking: Light-ASD, a small audio-visual active speaker detection
model (Liao et al., CVPR 2023, MIT license), run with ONNX Runtime
(docs/adr/0023).

The official weights are downloaded once from the authors' repository, pinned
to a commit and checked against a SHA-256, and read without PyTorch: the
checkpoint is a zip of raw tensors described by a pickle, which is read with
an unpickler that only accepts the four names such a file uses. The network
is then written as an ONNX graph in memory, with the batch norms folded into
the convolutions.

The model looks at a face, 112x112 gray, 25 frames a second, and at the
audio's MFCCs, 100 a second, and gives a score per frame: above 0, that face
is speaking.
"""

from __future__ import annotations

import hashlib
import os
import pickle
import tempfile
import threading
import urllib.request
import zipfile
from collections import OrderedDict
from pathlib import Path

import numpy as np

LIGHT_ASD_REPO = "https://raw.githubusercontent.com/Junhua-Liao/Light-ASD"
LIGHT_ASD_COMMIT = "ed38c232de5efe0261dbd68627c0ade7cdfe14eb"
# Fine-tuned on TalkSet (talking people in the wild), which suits interviews
# and podcasts better than the movie-trained weights.
LIGHT_ASD_FILE = "weight/finetuning_TalkSet.model"
LIGHT_ASD_SHA256 = "efc375833887eefa9d209dc92810e18519b04c3c73ea35a549f2a7f40b7d94d5"

FPS = 25  # video frames a second the model expects
MFCC_PER_FRAME = 4  # 100 MFCC frames a second
FACE_SIZE = 112
EPS = 0.001  # the model's batch norm epsilon
WINDOW = 100  # frames per front-end run (4 s), to bound memory
CONTEXT = 8  # frames of context on each side, more than the front ends see
DURATIONS = (1, 2, 3, 4, 5, 6)  # seconds per back-end chunk, averaged, as the authors do


def weights_path(cache_dir: str | os.PathLike) -> Path:
    return Path(cache_dir) / "light-asd" / f"{LIGHT_ASD_COMMIT[:12]}-{Path(LIGHT_ASD_FILE).name}"


def download_weights(cache_dir: str | os.PathLike, timeout: float = 60) -> Path:
    """The checkpoint in the cache, downloaded and checked on first use."""
    path = weights_path(cache_dir)
    if path.exists():
        return path
    if os.environ.get("HF_HUB_OFFLINE") == "1":
        raise RuntimeError(f"speaker model not in the cache ({path}) and downloads are off (HF_HUB_OFFLINE)")
    path.parent.mkdir(parents=True, exist_ok=True)
    url = f"{LIGHT_ASD_REPO}/{LIGHT_ASD_COMMIT}/{LIGHT_ASD_FILE}"
    with urllib.request.urlopen(url, timeout=timeout) as resp:  # a fixed https URL
        data = resp.read()
    digest = hashlib.sha256(data).hexdigest()
    if digest != LIGHT_ASD_SHA256:
        raise RuntimeError(f"speaker model checksum mismatch: got {digest}")
    with tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as tmp:
        tmp.write(data)
    os.chmod(tmp.name, 0o644)
    os.replace(tmp.name, path)
    return path


def read_checkpoint(path: str | os.PathLike) -> dict[str, np.ndarray]:
    """The tensors of a PyTorch checkpoint, as NumPy arrays, without PyTorch."""
    with zipfile.ZipFile(path) as z:
        root = z.namelist()[0].split("/")[0]
        storages: dict[str, np.ndarray] = {}

        def rebuild(storage, offset, size, stride, *_):
            if not size:
                return storage[offset].copy()
            strides = [s * storage.itemsize for s in stride]
            return np.lib.stride_tricks.as_strided(storage[offset:], shape=size, strides=strides).copy()

        allowed = {
            ("collections", "OrderedDict"): OrderedDict,
            ("torch._utils", "_rebuild_tensor_v2"): rebuild,
            ("torch", "FloatStorage"): np.dtype("<f4"),
            ("torch", "LongStorage"): np.dtype("<i8"),
        }

        class Unpickler(pickle.Unpickler):
            def find_class(self, module, name):
                if (module, name) not in allowed:
                    raise pickle.UnpicklingError(f"unexpected {module}.{name} in the checkpoint")
                return allowed[(module, name)]

            def persistent_load(self, pid):
                _, dtype, key, _location, _numel = pid
                if key not in storages:
                    storages[key] = np.frombuffer(z.read(f"{root}/data/{key}"), dtype=dtype)
                return storages[key]

        with z.open(f"{root}/data.pkl") as f:
            state = Unpickler(f).load()
    return {k: np.asarray(v) for k, v in state.items()}


# ---- The network as ONNX ----


class _Graph:
    def __init__(self, weights: dict[str, np.ndarray]):
        from onnx import helper, numpy_helper

        self.h, self.nh, self.w = helper, numpy_helper, weights
        self.nodes, self.inits, self.n = [], [], 0

    def name(self, prefix: str) -> str:
        self.n += 1
        return f"{prefix}_{self.n}"

    def const(self, value, prefix="c") -> str:
        name = self.name(prefix)
        self.inits.append(self.nh.from_array(np.asarray(value), name))
        return name

    def op(self, op_type: str, inputs: list[str], **attrs) -> str:
        out = self.name(op_type.lower())
        self.nodes.append(self.h.make_node(op_type, inputs, [out], **attrs))
        return out

    def conv_bn_relu(self, x: str, conv: str, bn: str, kernel, stride=None, pad=None) -> str:
        """Convolution then batch norm (folded into it) then ReLU."""
        w = self.w[f"{conv}.weight"].astype(np.float32)
        scale = self.w[f"{bn}.weight"] / np.sqrt(self.w[f"{bn}.running_var"] + EPS)
        bias = self.w[f"{bn}.bias"] - self.w[f"{bn}.running_mean"] * scale
        w = w * scale.reshape((-1,) + (1,) * (w.ndim - 1))
        dims = len(kernel)
        attrs = {"kernel_shape": list(kernel), "strides": list(stride or [1] * dims), "pads": list(pad or [0] * dims) * 2}
        y = self.op("Conv", [x, self.const(w.astype(np.float32), "w"), self.const(bias.astype(np.float32), "b")], **attrs)
        return self.op("Relu", [y])

    def block(self, x: str, prefix: str, spatial3, temporal3, spatial5, temporal5, down: bool, video: bool) -> str:
        """A Light-ASD block: two branches (3 and 5 wide), each a spatial (or
        frequency) then a temporal convolution, summed, then a 1x1."""
        def branch(k_s, k_t, s, t):
            stride = ([1, 2, 2] if down else None) if video else None
            pad_s = [k // 2 for k in k_s]
            pad_t = [k // 2 for k in k_t]
            y = self.conv_bn_relu(x, f"{prefix}.{s}", f"{prefix}.bn_{s}", k_s, stride, pad_s)
            return self.conv_bn_relu(y, f"{prefix}.{t}", f"{prefix}.bn_{t}", k_t, None, pad_t)

        y = self.op("Add", [branch(spatial3, temporal3, "s_3" if video else "m_3", "t_3"),
                            branch(spatial5, temporal5, "s_5" if video else "m_5", "t_5")])
        last = (1, 1, 1) if video else (1, 1)
        return self.conv_bn_relu(y, f"{prefix}.last", f"{prefix}.bn_last", last)

    def gru(self, x: str, prefix: str, direction: str) -> str:
        """PyTorch's GRU (gates r, z, n) as ONNX's (z, r, h), time first."""
        hidden = self.w[f"{prefix}.weight_hh_l0"].shape[1]

        def zrh(m):
            r, z, n = np.split(m, 3)
            return np.concatenate([z, r, n])

        w = zrh(self.w[f"{prefix}.weight_ih_l0"])[None].astype(np.float32)
        r = zrh(self.w[f"{prefix}.weight_hh_l0"])[None].astype(np.float32)
        b = np.concatenate([zrh(self.w[f"{prefix}.bias_ih_l0"]), zrh(self.w[f"{prefix}.bias_hh_l0"])])[None]
        out = self.name("gru")
        self.nodes.append(self.h.make_node(
            "GRU", [x, self.const(w), self.const(r), self.const(b.astype(np.float32))], [out],
            hidden_size=hidden, direction=direction, linear_before_reset=1,
        ))
        return self.op("Squeeze", [out, self.const(np.array([1], np.int64))])  # [T, 1, B, H] -> [T, B, H]

    def gelu(self, x: str) -> str:
        e = self.op("Erf", [self.op("Div", [x, self.const(np.float32(np.sqrt(2)))])])
        return self.op("Mul", [self.op("Mul", [x, self.const(np.float32(0.5))]), self.op("Add", [e, self.const(np.float32(1))])])

    def model(self, inputs, outputs) -> bytes:
        graph = self.h.make_graph(self.nodes, "light_asd", inputs, outputs, self.inits)
        model = self.h.make_model(graph, opset_imports=[self.h.make_opsetid("", 17)], producer_name="ultramedia")
        model.ir_version = 8
        return model.SerializeToString()


def frontend_onnx(weights: dict[str, np.ndarray]) -> bytes:
    """audio [1, 4T, 13] and faces [1, T, 112, 112] (0..255) -> [1, T, 128]."""
    from onnx import TensorProto

    g = _Graph(weights)
    h = g.h
    # Video: normalize, [1, 1, T, 112, 112], three blocks, max over the picture.
    v = g.op("Div", [g.op("Sub", [g.op("Div", ["faces", g.const(np.float32(255))]), g.const(np.float32(0.4161))]),
                     g.const(np.float32(0.1688))])
    v = g.op("Unsqueeze", [v, g.const(np.array([1], np.int64))])
    pool = {"kernel_shape": [1, 3, 3], "strides": [1, 2, 2], "pads": [0, 1, 1, 0, 1, 1]}
    v = g.block(v, "model.visualEncoder.block1", (1, 3, 3), (3, 1, 1), (1, 5, 5), (5, 1, 1), down=True, video=True)
    v = g.op("MaxPool", [v], **pool)
    v = g.block(v, "model.visualEncoder.block2", (1, 3, 3), (3, 1, 1), (1, 5, 5), (5, 1, 1), down=False, video=True)
    v = g.op("MaxPool", [v], **pool)
    v = g.block(v, "model.visualEncoder.block3", (1, 3, 3), (3, 1, 1), (1, 5, 5), (5, 1, 1), down=False, video=True)
    v = g.op("ReduceMax", [v], axes=[3, 4], keepdims=0)  # [1, 128, T]
    v = g.op("Transpose", [v], perm=[0, 2, 1])
    # Audio: [1, 1, 13, 4T], three blocks with time halved twice, mean over the coefficients.
    a = g.op("Transpose", [g.op("Unsqueeze", ["audio", g.const(np.array([1], np.int64))])], perm=[0, 1, 3, 2])
    pool = {"kernel_shape": [1, 3], "strides": [1, 2], "pads": [0, 1, 0, 1]}
    a = g.block(a, "model.audioEncoder.block1", (3, 1), (1, 3), (5, 1), (1, 5), down=False, video=False)
    a = g.op("MaxPool", [a], **pool)
    a = g.block(a, "model.audioEncoder.block2", (3, 1), (1, 3), (5, 1), (1, 5), down=False, video=False)
    a = g.op("MaxPool", [a], **pool)
    a = g.block(a, "model.audioEncoder.block3", (3, 1), (1, 3), (5, 1), (1, 5), down=False, video=False)
    a = g.op("Transpose", [g.op("ReduceMean", [a], axes=[2], keepdims=0)], perm=[0, 2, 1])
    g.op("Add", [a, v])
    g.nodes[-1].output[0] = "embedding"
    return g.model(
        [h.make_tensor_value_info("audio", TensorProto.FLOAT, [1, "A", 13]),
         h.make_tensor_value_info("faces", TensorProto.FLOAT, [1, "T", FACE_SIZE, FACE_SIZE])],
        [h.make_tensor_value_info("embedding", TensorProto.FLOAT, [1, "T", 128])],
    )


def backend_onnx(weights: dict[str, np.ndarray]) -> bytes:
    """embedding [1, T, 128] -> score [T]: a GRU each way, then the classifier."""
    from onnx import TensorProto

    g = _Graph(weights)
    h = g.h
    x = g.op("Transpose", ["embedding"], perm=[1, 0, 2])  # [T, 1, 128]
    x = g.gelu(g.gru(x, "model.GRU.gru_forward", "forward"))
    x = g.gelu(g.gru(x, "model.GRU.gru_backward", "reverse"))  # = flip, GRU, flip
    fc_w, fc_b = weights["lossAV.FC.weight"][1], weights["lossAV.FC.bias"][1]
    s = g.op("Add", [g.op("MatMul", [x, g.const(fc_w.astype(np.float32).reshape(128, 1))]), g.const(np.float32(fc_b))])
    g.op("Reshape", [s, g.const(np.array([-1], np.int64))])
    g.nodes[-1].output[0] = "score"
    return g.model(
        [h.make_tensor_value_info("embedding", TensorProto.FLOAT, [1, "T", 128])],
        [h.make_tensor_value_info("score", TensorProto.FLOAT, ["T"])],
    )


class SpeakerModel:
    """Light-ASD in ONNX Runtime. Thread-safe to load; one scorer at a time."""

    def __init__(self, cache_dir: str | os.PathLike):
        self.cache_dir = cache_dir
        self._sessions = None
        self._lock = threading.Lock()

    def load(self):
        with self._lock:
            if self._sessions is None:
                import onnxruntime as ort

                weights = read_checkpoint(download_weights(self.cache_dir))
                opts = ort.SessionOptions()
                opts.log_severity_level = 3
                providers = ["CPUExecutionProvider"]
                self._sessions = (
                    ort.InferenceSession(frontend_onnx(weights), opts, providers=providers),
                    ort.InferenceSession(backend_onnx(weights), opts, providers=providers),
                )
            return self._sessions

    def scores(self, mfcc: np.ndarray, faces: np.ndarray) -> np.ndarray:
        """Score per frame for one face: faces [T, 112, 112] uint8, mfcc
        [>= 4T, 13]. Above 0 means speaking."""
        _, backend = self.load()
        embedding = self.embed(mfcc, faces)
        t = embedding.shape[1]
        if t == 0:
            return np.zeros(0, np.float32)
        # Back end over chunks of several lengths, averaged.
        runs = []
        for seconds in DURATIONS:
            size = seconds * FPS
            run = [backend.run(None, {"embedding": embedding[:, i : i + size]})[0] for i in range(0, t, size)]
            runs.append(np.concatenate(run))
        return np.mean(runs, axis=0)

    def embed(self, mfcc: np.ndarray, faces: np.ndarray) -> np.ndarray:
        """The front ends' output, [1, T, 128], over windows with context on
        each side (more than their receptive field), as if run over the whole
        face at once but with bounded memory."""
        frontend, _ = self.load()
        t = min(len(faces), len(mfcc) // MFCC_PER_FRAME)
        if t == 0:
            return np.zeros((1, 0, 128), np.float32)
        faces = faces[:t].astype(np.float32)
        mfcc = mfcc[: t * MFCC_PER_FRAME].astype(np.float32)
        parts = []
        for start in range(0, t, WINDOW):
            end = min(start + WINDOW, t)
            a, b = max(0, start - CONTEXT), min(t, end + CONTEXT)
            (emb,) = frontend.run(None, {
                "audio": mfcc[a * MFCC_PER_FRAME : b * MFCC_PER_FRAME][None],
                "faces": faces[a:b][None],
            })
            parts.append(emb[0, start - a : start - a + (end - start)])
        return np.concatenate(parts)[None]


# ---- Audio features ----


def mfcc(signal: np.ndarray, rate: int = 16000) -> np.ndarray:
    """13 MFCCs every 10 ms over 25 ms windows, as python_speech_features
    computes them (the features Light-ASD was trained on): pre-emphasis 0.97,
    512-point FFT, 26 mel filters, log, DCT, lifter 22, and the first
    coefficient replaced by the log frame energy. signal is int16-scaled."""
    signal = np.asarray(signal, dtype=np.float64)
    signal = np.append(signal[:1], signal[1:] - 0.97 * signal[:-1])
    win, step, nfft = round(0.025 * rate), round(0.010 * rate), 512
    n = 1 if len(signal) <= win else 1 + int(np.ceil((len(signal) - win) / step))
    padded = np.concatenate([signal, np.zeros(max(0, (n - 1) * step + win - len(signal)))])
    idx = np.arange(win)[None, :] + step * np.arange(n)[:, None]
    frames = padded[idx]
    power = np.abs(np.fft.rfft(frames, nfft)) ** 2 / nfft
    energy = power.sum(axis=1)
    energy = np.where(energy == 0, np.finfo(float).eps, energy)
    feat = power @ _mel_filters(rate, nfft).T
    feat = np.log(np.where(feat == 0, np.finfo(float).eps, feat))
    feat = feat @ _dct_matrix(26)[:, :13]
    feat *= 1 + 11 * np.sin(np.pi * np.arange(13) / 22)
    feat[:, 0] = np.log(energy)
    return feat.astype(np.float32)


def _mel_filters(rate: int, nfft: int, count: int = 26) -> np.ndarray:
    def mel(hz):
        return 2595 * np.log10(1 + hz / 700.0)

    points = 700 * (10 ** (np.linspace(mel(0), mel(rate / 2), count + 2) / 2595.0) - 1)
    bins = np.floor((nfft + 1) * points / rate)
    bank = np.zeros((count, nfft // 2 + 1))
    for j in range(count):
        lo, mid, hi = int(bins[j]), int(bins[j + 1]), int(bins[j + 2])
        for i in range(lo, mid):
            bank[j, i] = (i - bins[j]) / (bins[j + 1] - bins[j])
        for i in range(mid, hi):
            bank[j, i] = (bins[j + 2] - i) / (bins[j + 2] - bins[j + 1])
    return bank


def _dct_matrix(n: int) -> np.ndarray:
    """Orthonormal DCT-II as a matrix: x @ M is scipy's dct(x, norm='ortho')."""
    k, i = np.meshgrid(np.arange(n), np.arange(n))
    m = np.cos(np.pi * k * (2 * i + 1) / (2 * n)) * np.sqrt(2 / n)
    m[:, 0] /= np.sqrt(2)
    return m
