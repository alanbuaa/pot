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
				//e.log.Infof("Decode byte:%s", hexutil.Encode(msgByte))
				continue
			}
			e.handlePacket(packet)
		}
	}
}

func (e *PoTEngine) handlePacket(packet *pb.Packet) {
	if packet.Type == pb.PacketType_P2PPACKET {
		if packet.ConsensusID == e.consensusID {
			potMsg := new(pb.PoTMessage)
			if err := proto.Unmarshal(packet.Msg, potMsg); err != nil {
				e.log.WithError(err).Warn("decode pot message failed")
				return
			}
			err := e.handlePoTMsg(potMsg)
			if err != nil {
				e.log.WithError(err).Warn("handle pot message error")
				return
			}
		} else {
			if e.UpperConsensus != nil && e.UpperConsensus.GetMsgByteEntrance() != nil {
				// TODO: cache message
				e.UpperConsensus.GetMsgByteEntrance() <- packet.GetMsg()
			}
		}
	} else if packet.Type == pb.PacketType_CLIENTPACKET {
		request := new(pb.Request)
		if err := proto.Unmarshal(packet.Msg, request); err != nil {
			e.log.WithError(err).Warn("unmarshal msg failed")
			return
		}

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
		pbHeader := new(pb.Header)
		err := proto.Unmarshal(bytes, pbHeader)
		if err != nil {
			return err
		}
		header := types.ToHeader(pbHeader)
		e.handleHeader(header)
	case pb.MessageType_Header_Request:
		bytes := message.GetMsgByte()
		request := new(pb.HeaderRequest)
		err := proto.Unmarshal(bytes, request)
		if err != nil {
			return err
		}
		// e.log.Infof("[Engine]\treceive header request from %s ", request.Src)
		hashes := request.GetHashes()
		st := e.GetHeaderStorage()
		header, err := st.Get(hashes)

		if err != nil {
			e.log.Errorf("get block err for %s", err)
			return err
		}
		pbHeader := header.ToProto()
		pbResponse := &pb.HeaderResponse{}
		if e.isBaseP2P {
			pbResponse = &pb.HeaderResponse{
				Header: pbHeader,
				Src:    e.peerId,
				Des:    request.GetSrc(),
				Srcid:  e.id,
				Desid:  request.GetSrcid(),
			}
		} else {
			pbResponse = &pb.HeaderResponse{
				Header: pbHeader,
				Src:    e.peerId,
				Des:    request.GetSrc(),
				Srcid:  e.id,
				Desid:  request.GetSrcid(),
			}
		}
		bytes, err = proto.Marshal(pbResponse)
		if err != nil {
			return err
		}
		potMsg := &pb.PoTMessage{
			MsgType: pb.MessageType_Header_Response,
			MsgByte: bytes,
		}
		msgByte, err := proto.Marshal(potMsg)
		if err != nil {
			return err
		}
		err = e.Unicast(request.GetSrc(), msgByte)
		if err != nil {
			return err
		}
	case pb.MessageType_Header_Response:
		// TODO: choose channel to put the response

		bytes := message.GetMsgByte()
		response := new(pb.HeaderResponse)
		err := proto.Unmarshal(bytes, response)
		// e.log.Infof("[Engine]\treceive header response from %s ", response.Src)
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
		e.log.Infof("[Engine]\treceive pot request from %s ", request.Src)
		epoch := request.GetEpoch()
		proof, err := e.headerStorage.GetPoTbyEpoch(epoch)
		if err != nil {
			return err
		}
		pbPoTResponse := &pb.PoTResponse{
			Epoch: epoch,
			Desid: request.Srcid,
			Des:   request.Src,
			Srcid: e.id,
			Src:   e.peerId,
			Proof: proof,
		}
		bytes, err = proto.Marshal(pbPoTResponse)
		if err != nil {
			return err
		}
		potMsg := &pb.PoTMessage{
			MsgType: pb.MessageType_PoT_Response,
			MsgByte: bytes,
		}
		msgByte, err := proto.Marshal(potMsg)
		if err != nil {
			return err
		}
		err = e.Unicast(request.GetSrc(), msgByte)
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
		e.log.Infof("[Engine]\treceive pot response from %s ", response.Src)
		err = e.worker.handlePoTResponse(response)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *PoTEngine) handleHeader(header *types.Header) {
	if header != nil {
		channel := e.worker.GetPeerQueue()
		channel <- header
	}

}

func (e *PoTEngine) broadcastHeader() {

}
