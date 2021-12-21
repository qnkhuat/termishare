const conn = new WebSocket("wss://server.termishare.com/ws");

const pc = new RTCPeerConnection({
    'iceServers': [{
        'urls': 'stun:stun.l.google.com:19302',
    }]
});

const sendChannel = pc.createDataChannel("sendChannel");
sendChannel.onopen = (e) => {
  console.log("data channel open");
  //setInterval(() => {

  //}, 500);
}

pc.onconnectionstatechange = (_e) => {
  console.log("Connection state change: ", pc.connectionState);
}


const handleMessage = (ev) => {
  const msg = JSON.parse(ev.data);
  console.log("Received a message:", msg);

  switch (msg.Type) {
    case "WillYouMarryMe":
      const offer = JSON.parse(msg.Data);
      console.log("set remote");
      pc.setRemoteDescription(offer);

      console.log("Create answer");
      const answer = pc.createAnswer().then((answer) => {

        const msg = JSON.stringify({
          Type: "Yes",
          Data: JSON.stringify(answer),
          To: "host",
          From: "Ngoc",
        })
        conn.send(msg);

        console.log("Set local");
        pc.setLocalDescription(answer);
      });

      break;

    case "Yes":
      console.log("Set remote");
      pc.setRemoteDescription(JSON.parse(msg.Data));
      break;

    case "Kiss":
      const candidate = new RTCIceCandidate(JSON.parse(msg.Data));
      console.log("add ice candidate")
      pc.addIceCandidate(candidate);
      break;
  }
}

pc.onicecandidate = (e) => {
  if (e.candidate) {
    console.log("Yes candidate");
    conn.send(JSON.stringify({
      To: "host",
      From: "Ngoc",
      Type: "Kiss",
      Data: JSON.stringify(e.candidate.toJSON())}))
  } else {
    console.log("No candidate");
  }
}

conn.onmessage = handleMessage;

conn.onopen = () => {
  console.log("Websocket connected!");
}

const clickToSend = async () => {
  // get media tracks
  //const stream = await navigator.mediaDevices.getUserMedia({audio:true});
  //const tracks = stream.getTracks()
  //tracks.forEach((track) => pc.addTrack(track));

  // send offer
  const offer = await pc.createOffer();
  console.log("Set local");
  pc.setLocalDescription(offer);
  const msg = JSON.stringify({
    Type: "WillYouMarryMe",
    Data: JSON.stringify(offer),
    To: "host",
    From: "Ngoc",
  })
  conn.send(msg);
}

pc.ondatachannel = e => {
  let dc = e.channel
  console.log('New DataChannel ' + dc.label)
  dc.onclose = () => console.log('dc has closed')
  dc.onopen = () => console.log('dc has opened')
  dc.onmessage = e => console.log(`Message from DataChannel '${dc.label}' payload '${e.data}'`)
  window.sendMessage = () => {
    let message = document.getElementById('message').value
    if (message === '') {
      return alert('Message must not be empty')
    }

    dc.send(message)
  }
}
