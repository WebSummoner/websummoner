---
title: Video recording
description: Record every browser session to an H.264 video file and manage the results.
---

WebSummoner can capture the browser screen of any session and save it as an
MPEG-4 (H.264) file. Recording works by attaching a small
[ffmpeg](https://www.ffmpeg.org/)-based container to the browser container —
so it works with any browser image, but only when browsers run in
containers.

The default recorder image (`websummoner/video-recorder:latest-release`) is
used automatically; pull it once on every host:

```bash
docker pull websummoner/video-recorder:latest-release
```

## Enabling recording

Add the `enableVideo` capability to the sessions you want recorded (all the
naming and tuning options are in the
[capabilities reference](/reference/capabilities/#video-recording-enablevideo-and-friends)):

```json
{ "enableVideo": true, "videoName": "checkout-flow.mp4" }
```

`videoName` must be a plain file name — no folders.

## Where files are stored

- **WebSummoner as a binary** — `./video` (override with `-video-output-dir`).
- **WebSummoner in Docker** — mount a host directory at
  `/opt/websummoner/video` **and** set `OVERRIDE_VIDEO_OUTPUT_DIR` to the
  same host path, so the recorder container writes to your storage:

  ```bash
  docker run -d --name websummoner \
    -p 4444:4444 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $PWD/config/:/etc/websummoner/:ro \
    -v $PWD/video/:/opt/websummoner/video/ \
    -e OVERRIDE_VIDEO_OUTPUT_DIR=$PWD/video/ \
    websummoner/websummoner:latest-release
  ```

  Why both the volume and the env var? The browser container writes the
  video, but it must land on **your** disk — see
  [bind mounts](https://docs.docker.com/engine/storage/bind-mounts/) for how
  host/container paths relate.

:::tip
[Configuration Manager](https://github.com/WebSummoner/cm) sets all of this
up automatically with `cm websummoner start`.
:::

## Downloading videos

Direct link (available after the session finishes — the temporary file is
renamed to `<session-id>.mp4` at session close):

```text
http://websummoner-host.example.com:4444/video/<filename>.mp4
```

List all files:

```text
http://websummoner-host.example.com:4444/video/
```

## Deleting videos

WebSummoner intentionally has no automatic cleanup — pick one:

**Scheduled deletion** — remove files older than 2 hours (Unix):

```bash
find /path/to/video/dir -mindepth 1 -maxdepth 1 -mmin +120 -name '*.mp4' | xargs rm -rf
```

**HTTP API** — delete from passed tests, for example:

```bash
curl -X DELETE http://websummoner-host.example.com:4444/video/<filename>.mp4
```

## Recording audio

The standard WebSummoner browser images run a PulseAudio server inside every
container. When the recorder can reach it (it connects over the container
network using a shared cookie), the mp4 automatically gets an **AAC audio
track** alongside the video — no flags or capabilities needed.

This means a session that plays media (a video site, an audio file, a
WebRTC call) is captured with sound. To opt out — for privacy, CPU or file
size reasons — set the `enableAudio: false`
[capability](/reference/capabilities/#video-recording-enablevideo-and-friends)
and the recording will be video-only.

To verify a recording contains audio:

```bash
ffprobe -show_entries stream=codec_type video.mp4   # look for codec_type=audio
ffmpeg -i video.mp4 -map 0:a -af volumedetect -f null -   # loudness stats
```

:::note
Browsers block autoplay **with sound** until a user gesture. To let tests
start playback programmatically, pass the matching browser flag, e.g. for
Chrome: `goog:chromeOptions: {"args": ["--autoplay-policy=no-user-gesture-required"]}`.
:::

## Choosing a codec

The default `libx264` gives the best compatibility. If recording consumes too
much CPU, switch per session with the `videoCodec` capability (e.g.
`"mpeg4"`). List everything your recorder supports:

```bash
docker run -it --rm --entrypoint /usr/bin/ffmpeg websummoner/video-recorder -codecs
```

## On Windows

Use a named volume or a converted path for the mounts, and keep
`OVERRIDE_VIDEO_OUTPUT_DIR` identical to the mounted host path:

```powershell
docker volume create websummoner-videos
$current = $PWD -replace "\\", "/" -replace "C", "c"
docker run -d --name websummoner `
  -p 4444:4444 `
  -v //var/run/docker.sock:/var/run/docker.sock `
  -v ${current}/config/:/etc/websummoner/:ro `
  -v /c/websummoner/video/:/opt/websummoner/video/ `
  -e OVERRIDE_VIDEO_OUTPUT_DIR=/c/websummoner/video/ `
  websummoner/websummoner:latest-release
```
