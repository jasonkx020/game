UPDATE companion_knowledge
SET content = '六子冲是 2 人棋类游戏，在 16 格棋盘上轮流落子，先连成六子者胜。'
WHERE game_id = 'liuzichong' AND title = '基本规则';

UPDATE companion_knowledge
SET content = '占据中心区域通常更有利；注意阻挡对手连线。'
WHERE game_id = 'liuzichong' AND title = '陪玩提示';
