/**
 * 战绩回放播放器骨架 — 完整实现待 P2 replay HTTP API。
 * @see docs/tech/replay.md
 */
export class ReplayNotImplementedError extends Error {
  constructor() {
    super('ReplayPlayer not implemented — waiting for P2 replay HTTP API')
    this.name = 'ReplayNotImplementedError'
  }
}

export interface ReplayPlayer {
  loadRound(roundId: string): Promise<void>
  play(): void
  pause(): void
  seek(actionSeq: number): void
}

export class StubReplayPlayer implements ReplayPlayer {
  async loadRound(_roundId: string): Promise<void> {
    throw new ReplayNotImplementedError()
  }
  play(): void {
    throw new ReplayNotImplementedError()
  }
  pause(): void {
    throw new ReplayNotImplementedError()
  }
  seek(_actionSeq: number): void {
    throw new ReplayNotImplementedError()
  }
}
