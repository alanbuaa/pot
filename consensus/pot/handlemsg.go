package pot

import (
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

func DecodePacket(packet []byte) (*pb.Packet, error) {
	p := new(pb.Packet)
	err := proto.Unmarshal(packet, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (e *PoTEngine) onReceiveMsg() {
	for {
		select {
		case msgByte, ok := <-e.MsgByteEntrance:
			if !ok {
				return
			}
			packet, err := DecodePacket(msgByte)
			if err != nil {
				e.log.WithError(err).Warn("decode packet failed")
				continue
			}
			e.handlePacket(packet)
		}
	}
}

func (e *PoTEngine) handlePacket(packet *pb.Packet) {
	if packet.Type == pb.PacketType_P2PPACKET {
		if packet.ConsensusID == e.consensusID {
			potmsg := new(pb.PoTMessage)
			if err := proto.Unmarshal(packet.Msg, potmsg); err != nil {
				e.log.WithError(err).Warn("decode pot message failed")
				return
			}
			err := e.handlePoTMsg(potmsg)
			if err != nil {
				e.log.WithError(err).Warn("handle pot message error")
				return
			}
		} else {
			if e.UpperConsensus != nil && e.UpperConsensus.GetMsgByteEntrance() != nil {
				e.UpperConsensus.GetMsgByteEntrance() <- packet.GetMsg()
			}
		}
	} else if packet.Type == pb.PacketType_CLIENTPACKET {
		msg := new(pb.Msg)
		if err := proto.Unmarshal(packet.Msg, msg); err != nil {
			e.log.WithError(err).Warn("unmarshal msg failed")
			return
		}
		request := msg.GetRequest()
		if request == nil {
			e.log.Warn("only request msg allowed in client packet")
			return
		}
		e.handleRequest(request)
	}
}

func (e *PoTEngine) handleRequest(request *pb.Request) {
	rtx := types.RawTransaction(request.Tx)
	if !e.exec.VerifyTx(rtx) {
		e.log.Warn("tx verify failed")
		return
	}
	tx, err := rtx.ToTx()
	if err != nil {
		e.log.WithError(err).Warn("decode into transaction failed")
		return
	}
	switch tx.Type {
	case pb.TransactionType_NORMAL:
		if e.UpperConsensus != nil {
			e.UpperConsensus.GetRequestEntrance() <- request
		}
	default:
		e.log.Warn("transaction type unknown", tx.Type.String())
	}
}

func (e *PoTEngine) handlePoTMsg(message *pb.PoTMessage) error {
	switch message.MsgType {
	case pb.MessageType_Header_Data:
		bytes := message.GetMsgByte()
		pbheader := new(pb.Header)
		err := proto.Unmarshal(bytes, pbheader)
		if err != nil {
			return err
		}
		header := types.ToHeader(pbheader)
		e.handleHeader(header)
	case pb.MessageType_Header_Request:
		bytes := message.GetMsgByte()
		request := new(pb.HeaderRequest)
		err := proto.Unmarshal(bytes, request)
		if err != nil {
			return err
		}
		hashes := request.GetHashes()
		st := e.GetHeaderStorage()
		header, err := st.Get(hashes)

		if err != nil {
			e.log.Errorf("get block err for %s", err)
			return err
		}
		pbheader := header.ToProto()
		pbresponse := &pb.HeaderResponse{}
		if e.isBasep2p {
			pbresponse = &pb.HeaderResponse{
				Header: pbheader,
				Src:    e.peerid,
				Des:    request.GetSrc(),
				Srcid:  e.id,
				Desid:  request.GetSrcid(),
			}
		} else {
			pbresponse = &pb.HeaderResponse{
				Header: pbheader,
				Src:    e.peerid,
				Des:    request.GetSrc(),
				Srcid:  e.id,
				Desid:  request.GetSrcid(),
			}
		}
		bytes, err = proto.Marshal(pbresponse)
		if err != nil {
			return err
		}
		potmsg := &pb.PoTMessage{
			MsgType: pb.MessageType_Header_Response,
			MsgByte: bytes,
		}
		msgbyte, err := proto.Marshal(potmsg)
		if err != nil {
			return err
		}
		err = e.Unicast(request.GetSrc(), msgbyte)
		if err != nil {
			return err
		}
	case pb.MessageType_Header_Response:
		// TODO: choose channel to put the response
		bytes := message.GetMsgByte()
		response := new(pb.HeaderResponse)
		err := proto.Unmarshal(bytes, response)
		if err != nil {
			return err
		}
		err = e.worker.handleHeaderResponse(response)
		if err != nil {
			return err
		}
	case pb.MessageType_PoT_Request:
		bytes := message.GetMsgByte()
		request := new(pb.PoTRequest)
		err := proto.Unmarshal(bytes, request)
		if err != nil {
			return err
		}
		epoch := request.GetEpoch()
		proof, err := e.headerStorage.GetPoTbyEpoch(epoch)
		if err != nil {
			return err
		}
		pbpotresponse := &pb.PoTResponse{
			Epoch: epoch,
			Desid: request.Srcid,
			Des:   request.Src,
			Srcid: e.id,
			Src:   e.peerid,
			Proof: proof,
		}
		bytes, err = proto.Marshal(pbpotresponse)
		if err != nil {
			return err
		}
		potmsg := &pb.PoTMessage{
			MsgType: pb.MessageType_PoT_Response,
			MsgByte: bytes,
		}
		msgbyte, err := proto.Marshal(potmsg)
		if err != nil {
			return err
		}
		err = e.Unicast(request.GetSrc(), msgbyte)
		if err != nil {
			return err
		}
	case pb.MessageType_PoT_Response:
		bytes := message.GetMsgByte()
		response := new(pb.PoTResponse)
		err := proto.Unmarshal(bytes, response)
		if err != nil {
			return err
		}
		err = e.worker.handlePoTResponse(response)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *PoTEngine) handleHeader(header *types.Header) {
	if header != nil {
		channel := e.worker.GetPeerqueue()
		channel <- header
	}

}

func (e *PoTEngine) braodcastHeader() {

}
