-- Where the main face is through each clip, for vertical exports that keep
-- it in frame (docs/adr/0022). NULL until the track-faces job has run:
-- {"coverage": 0.9, "points": [{"t": 0.0, "x": 0.52}], "width": 1920, "height": 1080}
-- t is seconds from the clip start; x is the centre of the frame to show,
-- as a share of the source width.
ALTER TABLE clips ADD COLUMN face_track JSONB;
