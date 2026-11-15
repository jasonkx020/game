INSERT INTO game_catalog (game_id, name, min_players, max_players, enabled)
VALUES ('dawugui', '打乌龟', 3, 5, true)
ON CONFLICT (game_id) DO NOTHING;

INSERT INTO game_ops_config (game_id, config)
VALUES ('dawugui', '{
  "room_card_cost": {"3": 2, "4": 2, "5": 3},
  "register_gift": {"game_coin": 5000, "room_card": 5},
  "coin_name": "龟币",
  "tiers": [
    {"tier_id":"novice","name":"新手场","base_score":100,"min_coin":5000},
    {"tier_id":"junior","name":"初级场","base_score":500,"min_coin":25000},
    {"tier_id":"mid","name":"中级场","base_score":2000,"min_coin":100000},
    {"tier_id":"senior","name":"高级场","base_score":10000,"min_coin":500000}
  ]
}'::jsonb)
ON CONFLICT (game_id) DO UPDATE SET config = EXCLUDED.config;

INSERT INTO users (phone, nickname, role)
VALUES ('13800000000', '平台管理员', 'platform_admin')
ON CONFLICT (phone) DO UPDATE SET role = 'platform_admin';

INSERT INTO wallet_room_card (user_id, balance)
SELECT id, 100 FROM users WHERE phone = '13800000000'
ON CONFLICT (user_id) DO UPDATE SET balance = 100;
