# 六子冲（民间六子棋）

> 规则原型：[docs/liuzichong.html](liuzichong.html)

## 概要

- **game_id**：`liuzichong`
- **人数**：2 人对战（seat 0 黑，seat 1 白）
- **棋盘**：4×4

## 规则

1. 黑方先行（seat 0）
2. 回合内选中己方棋子，移动到相邻空位（上下左右一步）
3. **吃子**：形成 `己-己-敌` 或 `敌-己-己` 三连，且两端外侧为空或边界
4. **胜负**：对方棋子 ≤1，或对方无合法步

## 协议

| 类型 | Route |
| :--- | :--- |
| C2S | `game.liuzichong.move` |
| Push | `onBoardInit`, `onTurnNotify`, `onMoveResult`, `onSettlement` |
