-- +goose Up
INSERT INTO levels (code, position, rating_min, rating_max, ielts_band_min, ielts_band_max) VALUES
  ('A1', 1,    0, 1000, 0.0, 2.5),
  ('A2', 2, 1000, 1200, 3.0, 4.0),
  ('B1', 3, 1200, 1400, 4.5, 5.5),
  ('B2', 4, 1400, 1600, 5.5, 6.5),
  ('C1', 5, 1600, 1800, 7.0, 8.0),
  ('C2', 6, 1800, 9999, 8.5, 9.0);

-- +goose Down
DELETE FROM levels;
