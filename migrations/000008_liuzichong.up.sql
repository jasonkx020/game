INSERT INTO game_catalog (game_id, name, min_players, max_players, enabled)
VALUES ('liuzichong', '六子冲', 2, 2, true)
ON CONFLICT (game_id) DO NOTHING;

INSERT INTO game_ops_config (game_id, config)
VALUES ('liuzichong', '{"room_card_cost":{"2":1}, "register_gift":{"game_coin":3000,"room_card":5}, "coin_name":"冲币"}')
ON CONFLICT (game_id) DO UPDATE SET config = EXCLUDED.config;
