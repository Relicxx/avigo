-- +goose Up
-- Индексы под горячие запросы.

-- Лента: фильтр по категории + сортировка по дате создания.
CREATE INDEX IF NOT EXISTS idx_listings_category ON listings (category);
CREATE INDEX IF NOT EXISTS idx_listings_created_at ON listings (created_at DESC);

-- Проверка активного буста и подзапрос сортировки ленты.
CREATE INDEX IF NOT EXISTS idx_boosts_listing_expires ON boosts (listing_id, expires_at);

-- Переписка по объявлению: выборка сообщений участника.
CREATE INDEX IF NOT EXISTS idx_messages_listing_sender ON messages (listing_id, sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_listing_receiver ON messages (listing_id, receiver_id);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_listing_receiver;
DROP INDEX IF EXISTS idx_messages_listing_sender;
DROP INDEX IF EXISTS idx_boosts_listing_expires;
DROP INDEX IF EXISTS idx_listings_created_at;
DROP INDEX IF EXISTS idx_listings_category;
