# termishare
Peer to peer terminal sharing

![ezgif com-gif-maker](https://user-images.githubusercontent.com/25661381/147855673-a30dd75e-b7d1-47b9-bdf8-2a6aeaf5df01.gif)

The motivation behind termishare is to provide a safe and fast way to access a remote terminal.

In order to achieve that, termishare uses a combination of WebSocket and WebRTC:
- WebSocket - is used only for signaling - which is a process to establish WebRTC connection
- [WebRTC](https://webrtc.org) - the primary connection to stream your terminal to other clients

## Getting started
1. Go to our [release](https://github.com/qnkhuat/termishare/releases) page and get a pre-built binary of `termishare`. Make sure you get the one that match your OS.
2. Untar the package `tar -xzf termishare_xyz.tar.gz`
3. (Optional) Move it to `/usr/local/bin` folder so that you could use `termishare` anywhere : `mv termishare /usr/local/bin`
4. Start sharing with `termishare`
    - If you don't want to connect to our TURN server, add a flag `-no-turn`
5. Done 🎉

### Note
There are chances where a direct peer-to-peer connection can't be established, so I included a TURN server that I created using [CoTURN](https://github.com/coturn/coturn).

If you want to ensure your terminal is not connected to an unknown server, you can:
- Disable the usage of turn server (with `-no-turn` flag)
- Creates your own TURN server connect to it by changing in [cfg/termishare.go](cli/internal/cfg/server.go)

## Upcoming
- [ ] Connect to termishare session via `termishare` itself, instead of web-client
- [ ] Install via brew/apt
- [ ] Approval mechanism
