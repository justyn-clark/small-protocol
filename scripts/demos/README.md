# SMALL release demos

The v1.1.0 demo is a real terminal recording, not a simulated transcript. It starts from a clean, SMALL-enabled Git project and uses the released `small v1.1.0` binary to:

- verify the existing canonical state;
- preview and apply the explicit `2.0.0` migration;
- confirm solo mode as the default;
- start a session and opt in to collaborative mode; and
- finish with another strict validation pass.

The source project is copied with local, no-hardlink clones. The original checkout is read-only for the recording and remains unchanged.

## Render

Install `vhs`, `ffmpeg`, `git`, and `jq`, then run:

```sh
SMALL_BIN=/path/to/small-v1.1.0 \
  ./scripts/demos/render-small-v1-1-0-session-demo.sh \
  /path/to/clean/small-enabled-project
```

The renderer requires the exact `small v1.1.0` release binary and a clean source checkout. It validates both before capture.

Outputs:

- `docs/media/small-v1-1-0-session-demo.mp4` — H.264 for social platforms and broad browser support
- `docs/media/small-v1-1-0-session-demo.webm` — VP9 web version
- `docs/media/small-v1-1-0-session-demo-poster.png` — first-frame poster

VHS 0.12.0 currently reports success even when its direct video encoder fails on this macOS setup. The renderer therefore asks VHS for the real text and cursor frame sequences, composites those frames with FFmpeg, and verifies the resulting codecs.
