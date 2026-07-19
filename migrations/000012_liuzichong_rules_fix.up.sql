-- Fix liuzichong companion knowledge to match real rules (冲吃, not 连成六子)
UPDATE companion_knowledge
SET content = '六子冲是 2 人棋类游戏，4×4 棋盘上正交走一步。形成「己-己-敌」或「敌-己-己」且敌方外侧为空（边界算空）则吃子。对方棋子≤1 或无合法步即胜。'
WHERE game_id = 'liuzichong' AND title = '基本规则';

UPDATE companion_knowledge
SET content = '优先制造冲吃；注意敌方外侧是否为空。边角常是冲吃关键。'
WHERE game_id = 'liuzichong' AND title = '陪玩提示';
