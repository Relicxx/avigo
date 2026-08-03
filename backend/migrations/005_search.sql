-- +goose Up
-- Полнотекстовый поиск по объявлениям (title + description).
-- Конфигурация 'simple': заголовки и описания смешивают русский и английский,
-- а языкозависимый стемминг одного языка исказил бы токены другого.
ALTER TABLE listings ADD COLUMN IF NOT EXISTS search_tsv tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', title || ' ' || coalesce(description, ''))
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_listings_search ON listings USING GIN (search_tsv);

-- +goose Down
DROP INDEX IF EXISTS idx_listings_search;
ALTER TABLE listings DROP COLUMN IF EXISTS search_tsv;
