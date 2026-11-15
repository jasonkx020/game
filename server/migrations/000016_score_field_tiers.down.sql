UPDATE game_ops_config
SET config = config - 'tiers',
    updated_at = now()
WHERE game_id = 'dawugui';
