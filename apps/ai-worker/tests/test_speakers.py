"""The speaker model: audio features, the safe checkpoint reader, and the
ONNX network with the official weights (downloaded once, skipped offline)."""

import io
import os
import pickle
import zipfile

import numpy as np
import pytest

from ultramedia_ai import speakers


def sine(seconds=1.0, hz=440, amplitude=10000):
    t = np.arange(int(16000 * seconds)) / 16000
    return (amplitude * np.sin(2 * np.pi * hz * t)).astype(np.int16)


# ---- Audio features ----


def test_mfcc_matches_python_speech_features():
    # Reference rows from python_speech_features.mfcc(sig, 16000, numcep=13,
    # winlen=0.025, winstep=0.010), the features Light-ASD was trained on.
    m = speakers.mfcc(sine())
    assert m.shape == (99, 13)
    assert m[0] == pytest.approx([19.5074, 18.8076, 1.7124, -14.1897, -27.1832, -33.5303, -29.6864, -18.5563,
                                  -3.2681, 10.0441, 18.0148, 17.9862, 11.9377], abs=1e-3)
    assert m[50] == pytest.approx([19.5121, 19.5095, 2.6441, -13.1919, -26.1621, -32.5259, -28.6832, -17.6791,
                                   -2.3542, 10.7892, 18.8813, 18.6872, 12.3887], abs=1e-3)


def test_mfcc_of_silence_is_finite():
    m = speakers.mfcc(np.zeros(1600, np.int16))
    assert np.isfinite(m).all()


# ---- Checkpoint reader ----


class Evil:
    def __reduce__(self):
        return (os.system, ("echo pwned",))


def test_the_checkpoint_reader_refuses_anything_but_tensors(tmp_path):
    path = tmp_path / "evil.model"
    with zipfile.ZipFile(path, "w") as z:
        z.writestr("archive/data.pkl", pickle.dumps({"x": Evil()}))
    with pytest.raises(pickle.UnpicklingError, match="unexpected"):
        speakers.read_checkpoint(path)


def test_downloads_are_checked(tmp_path, monkeypatch):
    class Response(io.BytesIO):
        def __enter__(self):
            return self

        def __exit__(self, *a):
            return False

    monkeypatch.delenv("HF_HUB_OFFLINE", raising=False)
    monkeypatch.setattr(speakers.urllib.request, "urlopen", lambda url, timeout: Response(b"not the model"))
    with pytest.raises(RuntimeError, match="checksum"):
        speakers.download_weights(tmp_path)
    assert not speakers.weights_path(tmp_path).exists()


# ---- The network ----


@pytest.fixture(scope="module")
def model():
    from huggingface_hub import constants

    m = speakers.SpeakerModel(constants.HF_HOME)
    try:
        m.load()
    except Exception as e:  # noqa: BLE001
        pytest.skip(f"speaker model not available: {e}")
    return m


def test_the_official_weights_are_read(model):
    weights = speakers.read_checkpoint(speakers.weights_path(model.cache_dir))
    assert sum(v.size for k, v in weights.items() if k.startswith("model.")) == 1_025_630  # 1.0 M, as published
    assert weights["lossAV.FC.weight"].shape == (2, 128)


def test_windows_give_the_same_embedding_as_one_run(model):
    rng = np.random.default_rng(0)
    faces = rng.integers(0, 255, (230, 112, 112)).astype(np.uint8)
    mfcc = speakers.mfcc(rng.normal(0, 3000, 16000 * 10).astype(np.int16))
    frontend, _ = model.load()
    (whole,) = frontend.run(None, {"audio": mfcc[None, : 230 * 4], "faces": faces[None].astype(np.float32)})
    windowed = model.embed(mfcc, faces)
    assert windowed.shape == (1, 230, 128)
    np.testing.assert_allclose(windowed, whole, atol=1e-3)


def test_a_still_gray_face_in_silence_is_not_speaking(model):
    scores = model.scores(speakers.mfcc(np.zeros(16000 * 2 + 400, np.int16)), np.full((50, 112, 112), 110, np.uint8))
    assert scores.shape == (50,) and (scores < 0).all()
