-- +goose Up
CREATE TABLE vocabulary_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 terms TEXT NOT NULL CHECK(length(CAST(terms AS BLOB))<=16384),
 voice INTEGER NOT NULL CHECK(voice IN (0,1)),
 files INTEGER NOT NULL CHECK(files IN (0,1)),
 boost REAL NOT NULL CHECK(boost BETWEEN 0 AND 5)
) STRICT;
-- Preserve all active text. Newline-separated phrases are normalized by Go.
INSERT INTO vocabulary_settings
SELECT 1, v.vocabulary || char(10) || v.hotwords || char(10) || t.transcription_options_hotwords,
 length(trim(v.vocabulary || v.hotwords))>0, length(trim(t.transcription_options_hotwords))>0,
 CASE WHEN length(trim(v.vocabulary))>0 THEN v.boost ELSE 3 END
FROM voice_transcription_settings v CROSS JOIN transcription_settings t WHERE v.id=1 AND t.id=1;
INSERT OR IGNORE INTO vocabulary_settings VALUES(1,'',0,0,3);
UPDATE voice_transcription_settings SET vocabulary='',hotwords='';
UPDATE transcription_settings SET transcription_options_hotwords='';
-- Historical model rows remain intact; selection no longer restores their terms.
