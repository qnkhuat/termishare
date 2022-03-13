# termishare
Peer to peer terminal sharing

![ezgif com-gif-maker-2](https://user-images.githubusercontent.com/25661381/150143689-778b4f02-8787-42e6-a5a1-03809b62c4f4.gif)

The motivation behind termishare is to provide a safe and fast way to access a remote terminal.

In order to achieve that, termishare uses a combination of WebSocket and WebRTC:
- WebSocket - is used only for signaling - which is a process to establish WebRTC connection
- [WebRTC](https://webrtc.org) - the primary connection to stream your terminal to other clients

# Getting started

## Install
### Using brew
`brew tap qnkhuat/tap && brew install termishare`

### Using release

1. Go to our [release](https://github.com/qnkhuat/termishare/releases) page and get a pre-built binary of `termishare`. Make sure you get the one that match your OS.
2. Untar the package `tar -xzf termishare_x.y.z.tar.gz`
3. (Optional) Move it to `/usr/local/bin` folder so that you could use `termishare` anywhere : `mv termishare /usr/local/bin`

## Usage
1. To start a sharing session, just run `termisnare`
2. Termishare will echo out a connection url you can use to connect via:
    - browser
    - terminal: `termishare {{connection_url}}`

### Note
There are chances where a direct peer-to-peer connection can't be established, so I included a TURN server that I created using [CoTURN](https://github.com/coturn/coturn).

If relay to the TURN server is something you don't want, you can:
- Disable the usage of turn server (with `-no-turn` flag)
- Creates your own TURN server connect to it by changing in [cfg/termishare.go](cli/internal/cfg/server.go) then re-compile termishare (sorry)

## Self-hosted
Termishare server is a jar file, it contains both the signaling server and the UI, so it's fairlly simple to self-host termishare:
1. Install java
2. Download `termishare.jar` from our [release](https://github.com/qnkhuat/termishare/releases) page
3. Start it with `java -jar termishare.jar`
4. Now you can connect to your server using termishare with `termishare -server {{sever_address}}`

## Upcoming
- [x] Move both the front-end and server to server as one
- [x] Connect to termishare session via `termishare` itself, instead of web-client
- [x] Install via brew/apt
- [ ] Customize TURN server
- [ ] Approval mechanism

## Similar projects
- https://github.com/elisescu/tty-share
- https://github.com/tmate-io/tmate
