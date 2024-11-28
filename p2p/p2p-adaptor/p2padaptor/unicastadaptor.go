package p2padaptor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	"io"
	net "network"

	libp2pnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"encoding/binary"
)

var (
	protocalID = "/consensus/1.0.0"
)

type UnicastAdapter struct {
	p2pnet   *net.Network
	entrance chan<- []byte
	streams  map[peer.ID]libp2pnet.Stream
}

func NewUniCastAdaptor(p2pnet *net.Network) (*UnicastAdapter, error) {

	return &UnicastAdapter{
		p2pnet:  p2pnet,
		streams: make(map[peer.ID]libp2pnet.Stream),
	}, nil
}

func (uca *UnicastAdapter) StartUnicastService() {
	nethost := uca.p2pnet.GetHost()

	handleStreamFunc := func(stream libp2pnet.Stream) {
		go func(stream libp2pnet.Stream) {
			for {
				if stream.Conn().IsClosed() {
					return
				}

				// Read header
				buf := bufio.NewReader(stream)
				header0, err := buf.ReadByte()
				if err != nil {
					log.Println("Error reading header0 from buffer")
					return
				}

				header1, err := buf.ReadByte()
				if err != nil {
					log.Println("Error reading header1 from buffer")
					return
				}

				header2, err := buf.ReadByte()
				if err != nil {
					log.Println("Error reading header2 from buffer")
					return
				}

				header3, err := buf.ReadByte()
				if err != nil {
					log.Println("Error reading header3 from buffer")
					return
				}

				var header []byte
				header = append(header, header0)
				header = append(header, header1)
				header = append(header, header2)
				header = append(header, header3)

				// Read payload
				payload := make([]byte, BytesToInt(header))
				_, err = io.ReadFull(buf, payload)
				// fmt.Printf("payload has %d bytes", n)
				if err != nil {
					log.Println("Error reading from buffer")
					return
				}

				// fmt.Printf("read: %s", payload)

				if uca.entrance != nil {
					uca.entrance <- payload
				}
			}
		}(stream)
	}

	nethost.SetStreamHandler(protocol.ID(protocalID), handleStreamFunc)
}

func (uca *UnicastAdapter) SendUnicast(address string, msgByte []byte, consensusID int64) error {
	// Whether the address is a peer.id string?
	peerID, err := peer.Decode(address)
	if err != nil {
		return fmt.Errorf("decode peerid error, %s is not a peerid string", address)
	}

	// Whether to connect with this peer?
	if !(uca.p2pnet.GetHost().Network().Connectedness(peerID) == libp2pnet.Connected) {
		ctx := context.Background()
		nethost := uca.p2pnet.GetHost()

		// Find peer
		addr, err := uca.p2pnet.FindPeerWithID(peerID)
		if err != nil {
			return errors.New("can not find this peer")
		}

		// Connect peer
		if err := nethost.Connect(ctx, addr); err != nil {
			return errors.New("can not connect this peer")
		}
	}

	// Does the stream exist？
	mystream, ok1 := uca.streams[peerID]
	if !ok1 {
		// new
		mystream, err = uca.newStream(peerID)
		if err != nil {
			return errors.New("can not build stream")
		}
		// store
		uca.streams[peerID] = mystream
	}

	// Whether the stream is closed?
	if mystream.Conn().IsClosed() {
		// close
		mystream.Close()
		delete(uca.streams, peerID)

		// new
		var err error
		mystream, err = uca.newStream(peerID)
		if err != nil {
			return errors.New("can not build stream")
		}

		// store
		uca.streams[peerID] = mystream
	}

	err = uca.sendBytes(mystream, msgByte)
	if err != nil {
		return errors.New("send bytes error")
	}

	return nil
}

func (uca *UnicastAdapter) newStream(peerID peer.ID) (libp2pnet.Stream, error) {
	// New stream
	ctx := context.Background()
	nethost := uca.p2pnet.GetHost()

	// Stream
	st, err := nethost.NewStream(ctx, peerID, protocol.ID(protocalID))
	if err != nil {
		return st, errors.New("new stream open failed")
	}

	uca.streams[peerID] = st
	return st, nil
}

func (uca *UnicastAdapter) sendBytes(st libp2pnet.Stream, data []byte) error {
	payload := data
	header := IntToBytes(len(payload))

	// Write header
	_, err := st.Write(header)
	if err != nil {
		return errors.New("error write header")
	}

	// Write payload
	_, err = st.Write(payload)
	if err != nil {
		return errors.New("error write playload")
	}

	return nil
}

func IntToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}
