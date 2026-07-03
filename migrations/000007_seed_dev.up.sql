INSERT INTO game_catalog (game_id, name, min_players, max_players, enabled)
VALUES ('dawugui', '打乌龟', 3, 5, true)
ON CONFLICT (game_id) DO NOTHING;

INSERT INTO game_ops_config (game_id, config)
VALUES ('dawugui', '{
  "room_card_cost": {"3": 2, "4": 2, "5": 3},
  "register_gift": {"game_coin": 5000, "room_card": 5},
  "coin_name": "龟币"
}'::jsonb)
ON CONFLICT (game_id) DO UPDATE SET config = EXCLUDED.config;

INSERT INTO users (phone, nickname, role)
VALUES ('13800000000', '平台管理员', 'platform_admin')
ON CONFLICT (phone) DO UPDATE SET role = 'platform_admin';

INSERT INTO wallet_room_card (user_id, balance)
SELECT id, 100 FROM users WHERE phone = '13800000000'
ON CONFLICT (user_id) DO UPDATE SET balance = 100;
