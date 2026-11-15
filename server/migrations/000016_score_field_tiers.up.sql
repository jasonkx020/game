-- Sync dawugui ops-hooks with score-field tiers (idempotent merge into config).
UPDATE game_ops_config
SET config = config || '{
  "tiers": [
    {"tier_id":"novice","name":"新手场","base_score":100,"min_coin":5000},
    {"tier_id":"junior","name":"初级场","base_score":500,"min_coin":25000},
    {"tier_id":"mid","name":"中级场","base_score":2000,"min_coin":100000},
    {"tier_id":"senior","name":"高级场","base_score":10000,"min_coin":500000}
  ]
}'::jsonb,
    updated_at = now()
WHERE game_id = 'dawugui';
