CREATE EXTENSION IF NOT EXISTS postgis;

-- activities
CREATE TABLE IF NOT EXISTS activities (
  id UUID PRIMARY KEY,
  started_at TIMESTAMPTZ NOT NULL,
  sport TEXT NULL,
  duration_sec INT NULL,
  distance_m INT NULL,
  avg_hr SMALLINT NULL,
  max_hr SMALLINT NULL,
  bounds GEOGRAPHY(POLYGON,4326) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- trackpoints
CREATE TABLE IF NOT EXISTS trackpoints (
  id BIGSERIAL PRIMARY KEY,
  activity_id UUID REFERENCES activities(id) ON DELETE CASCADE,
  t TIMESTAMPTZ NOT NULL,
  ele_m REAL NULL,
  hr SMALLINT NULL,
  speed_mps REAL NULL,
  geom GEOGRAPHY(POINT,4326) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tp_activity_time ON trackpoints(activity_id, t);
CREATE INDEX IF NOT EXISTS idx_tp_geom ON trackpoints USING GIST(geom);
