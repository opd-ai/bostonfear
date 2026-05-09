# Boolean and Control Flow Logic Gaps — 2026-05-09

## Reconnection Reliability Gap (Ebitengine Client)
- **Stated Goal**: Client reconnects automatically with exponential backoff after failure.
- **Current State**: In client/ebiten/net.go, readLoop waits on done after read error, but writeLoop only exits on actionsCh activity or write error; with no queued action, done is never closed and reconnectLoop is never resumed.
- **Risk**: Idle disconnection causes permanent client disconnect (wrong branch progression: stuck waiting instead of retrying).
- **Closing the Gap**: Add shutdown signaling branch in writeLoop (case <-done) and make read error path actively signal writer shutdown before waiting. Add regression test: read error + empty actionsCh must return from runConnection within timeout.

## Turn Progression Gap When All Connected Players Are Defeated
- **Stated Goal**: Turn progression should never stall and defeated investigators recover in Mythos phase.
- **Current State**: advanceTurn only progresses when it finds exists && connected && !defeated; if all connected players are defeated, no branch executes and function returns with no recovery or phase transition.
- **Risk**: Playing phase can deadlock: no legal acting player and no automatic path to Mythos recovery.
- **Closing the Gap**: Add explicit no-active-candidate branch after loop: trigger Mythos recovery or transition to deterministic waiting state, then re-evaluate current player. Add test for all-connected-defeated scenario that fails pre-fix and passes post-fix.
