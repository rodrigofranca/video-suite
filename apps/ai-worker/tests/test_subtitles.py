from ultramedia_ai.subtitles import Cue, Segment, Word, build_cues, timestamp, to_webvtt, wrap


def test_timestamp():
    assert timestamp(0) == "00:00:00.000"
    assert timestamp(62.5) == "00:01:02.500"
    assert timestamp(3723.0456) == "01:02:03.046"


def test_short_segment_is_one_cue():
    seg = Segment(0, 2, "Hello there.", [Word(0, 0.5, " Hello"), Word(0.6, 1.2, " there.")])
    assert build_cues([seg]) == [Cue(0, 1.2, "Hello there.")]


def test_long_segment_is_split_by_time_and_length():
    words = [Word(i, i + 0.8, f" word{i}") for i in range(20)]  # 20 seconds of speech
    cues = build_cues([Segment(0, 20, "", words)])
    assert len(cues) > 1
    for cue in cues:
        assert cue.end - cue.start <= 6.0
        assert len(cue.text) <= 84
    assert " ".join(c.text for c in cues) == " ".join(w.text.strip() for w in words)


def test_segment_without_words_is_kept():
    assert build_cues([Segment(1, 3, " no word timings ")]) == [Cue(1, 3, "no word timings")]


def test_wrap_breaks_near_the_middle():
    text = "this sentence is long enough to need two lines on screen"
    lines = wrap(text).split("\n")
    assert len(lines) == 2 and all(len(line) <= 42 for line in lines)
    assert wrap("short line") == "short line"


def test_webvtt_document():
    vtt = to_webvtt([Cue(0, 1.5, "Hi --> there"), Cue(2, 2, "zero length")])
    assert vtt.startswith("WEBVTT\n\n1\n00:00:00.000 --> 00:00:01.500\nHi → there\n")
    assert "00:00:02.000 --> 00:00:02.100" in vtt  # never ends before it starts
