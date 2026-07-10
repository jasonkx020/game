# 打乌龟独立 App 构建说明（POC）
# 1. 在 Cocos 中打开 GameS/client，构建目标仅含 dawugui Bundle + StandaloneEntry
# 2. 包名 com.games.dawugui；Deep link: dawugui://play
# 3. Standalone 冷启动调用 assets/games/dawugui/StandaloneEntry.ts

param(
    [string]$GameId = "dawugui"
)

Write-Host "Standalone POC: game=$GameId"
Write-Host "Entry: client/assets/games/$GameId/StandaloneEntry.ts"
Write-Host "Meta: game_standalone table (deep_link_scheme, min_host_version)"
Write-Host "Run Cocos build manually or CI pipeline for APK output."
