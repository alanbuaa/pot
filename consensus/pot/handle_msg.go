package pot

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

func (e *PoTEngine) onReceiveMsg() {
	for {
		select {
		case msgByte, ok := <-e.MsgByteEntrance:
			if !ok {
				return
			}
			packet, err := DecodePacket(msgByte)
			if err != nil {
				e.log.WithError(err).Warn("Failed to decode packet from network")
				e.log.WithField("raw_data", hexutil.Encode(msgByte)).Trace("Invalid packet data (hex dump)")
				continue
			}
			e.handlePacket(packet)
		}
	}
}

// type ControllerMessage struct {
// 	Data     []byte
// 	Receiver string
// }

func (e *PoTEngine) handlePacket(packet *pb.Packet) {
	e.log.WithFields(logrus.Fields{
		"packet_type":  packet.Type.String(),
		"consensus_id": packet.ConsensusID,
	}).Trace("[TRACE-2] PoTEngine received packet")

	if packet.Type == pb.PacketType_P2PPACKET {
		if packet.ConsensusID == e.consensusID {
			e.log.Trace("[TRACE-2.1] Handling PoT internal message")
			potMsg := new(pb.PoTMessage)
			if err := proto.Unmarshal(packet.Msg, potMsg); err != nil {
				e.log.WithError(err).Warn("Failed to unmarshal PoT message")
				return
			}
			err := e.handlePoTMsg(potMsg)
			if err != nil {
				e.log.WithError(err).Warn("Failed to handle PoT message")
				return
			}
		} else {
			if e.UpperConsensus != nil && e.UpperConsensus.GetMsgByteEntrance() != nil {
				e.log.WithField("target_consensus_id", packet.ConsensusID).Trace("[TRACE-2.2] Forwarding P2P packet to upper consensus (Whirly)")
				// TODO: cache message
				// e.UpperConsensus.GetMsgByteEntrance() <- packet.GetMsg()
				// controllerMessage := &ControllerMessage{
				// 	Data:     packet.GetMsg(),
				// 	Receiver: packet.ReceiverPublicAddress,
				// }
				// controllerMessageBytes, err := json.Marshal(controllerMessage)
				// if err != nil {
				// 	e.log.WithError(err).Warn("encode controllerMessage failed")
				// 	return
				// }
				// e.UpperConsensus.GetMsgByteEntrance() <- controllerMessageBytes
				bytePacket, err := proto.Marshal(packet)
				if err != nil {
					e.log.WithError(err).Warn("Failed to marshal packet")
					return
				}
				e.UpperConsensus.GetMsgByteEntrance() <- bytePacket
			}
		}
	} else if packet.Type == pb.PacketType_CLIENTPACKET {
		e.log.Trace("[TRACE-2.3] Received CLIENTPACKET, unmarshalling request")
		request := new(pb.Request)
		if err := proto.Unmarshal(packet.Msg, request); err != nil {
			e.log.WithError(err).Error("Failed to unmarshal client request")
			return
		}

		if request == nil {
			e.log.Error("Received null request in client packet")
			return
		}
		e.log.WithField("sharding", string(request.Sharding)).Trace("[TRACE-2.4] Calling handleRequest")
		e.handleRequest(request)
	}
}

func (e *PoTEngine) handleRequest(request *pb.Request) {
	e.log.Trace("[TRACE-3] handleRequest called")
	rtx := types.RawTransaction(request.Tx)
	txHash := rtx.Hash()
	e.log.WithField("tx_hash", hexutil.Encode(txHash[:8])).Debug("Verifying transaction")

	if !e.exec.VerifyTx(rtx) {
		e.log.WithField("tx_hash", txHash).Error("[TRACE-3.1] Transaction verification FAILED")
		return
	}
	e.log.Trace("[TRACE-3.2] Transaction verification passed")

	tx, err := rtx.ToTx()
	if err != nil {
		e.log.WithError(err).Error("[TRACE-3.3] Failed to decode transaction payload")
		return
	}

	e.log.WithFields(logrus.Fields{
		"type":            tx.Type.String(),
		"payload_preview": string(tx.Payload[:min(len(tx.Payload), 50)]),
	}).Trace("[TRACE-3.4] Processing transaction")

	switch tx.Type {
	case pb.TransactionType_NORMAL:
		if e.UpperConsensus != nil {
			e.log.Trace("[TRACE-3.5] Forwarding NORMAL transaction to UpperConsensus (Whirly)")
			e.UpperConsensus.GetRequestEntrance() <- request
			e.log.Trace("[TRACE-3.6] Transaction forwarded to Whirly successfully")
		} else {
			e.log.Error("[TRACE-3.5-ERROR] UpperConsensus is nil, cannot forward transaction!")
		}
	default:
		e.log.WithField("type", tx.Type.String()).Warn("Unknown transaction type")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (e *PoTEngine) handlePoTMsg(message *pb.PoTMessage) error {
	switch message.MsgType {
	case pb.MessageType_Block_Data:
		bytes := message.GetMsgByte()
		pbHeader := new(pb.Block)
		err := proto.Unmarshal(bytes, pbHeader)
		if err != nil {
			return err
		}
		b := types.ToBlock(pbHeader)
		e.handleblock(b)
	case pb.MessageType_Block_Request:
		bytes := message.GetMsgByte()
		request := new(pb.BlockRequest)
		err := proto.Unmarshal(bytes, request)
		if err != nil {
			return err
		}
		// e.log.Infof("[Engine]\treceive block request from %s ", request.Src)
		hashes := request.GetHashes()
		st := e.GetBlockStorage()
		block, err := st.Get(hashes)

		if err != nil {
			e.log.WithError(err).WithField("block_hash", hexutil.Encode(hashes)).Error("Failed to get requested block")
			return err
		}
		pbblock := block.ToProto()
		pbResponse := &pb.BlockResponse{}
		if e.isBaseP2P {
			pbResponse = &pb.BlockResponse{
				Block: pbblock,
				Src:   e.peerId,
				Des:   request.GetSrc(),
				Srcid: e.id,
				Desid: request.GetSrcid(),
			}
		} else {
			pbResponse = &pb.BlockResponse{
				Block: pbblock,
				Src:   e.peerId,
				Des:   request.GetSrc(),
				Srcid: e.id,
				Desid: request.GetSrcid(),
			}
		}
		bytes, err = proto.Marshal(pbResponse)
		if err != nil {
			return err
		}
		potMsg := &pb.PoTMessage{
			MsgType: pb.MessageType_Block_Response,
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
	case pb.MessageType_Block_Response:
		// TODO: choose channel to put the response

		bytes := message.GetMsgByte()
		response := new(pb.BlockResponse)
		err := proto.Unmarshal(bytes, response)

		if err != nil {
			return err
		}
		err = e.Worker.handleBlockResponse(response)
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
		e.log.WithFields(logrus.Fields{
			"from":  request.Src,
			"epoch": request.GetEpoch(),
		}).Trace("Received PoT request")
		epoch := request.GetEpoch()
		proof, err := e.blockStorage.GetVDFresbyEpoch(epoch)
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
		//e.log.Infof("[Engine]\treceive pot response from %s ", response.Src)
		err = e.Worker.handlePoTResponse(response)
		if err != nil {
			return err
		}
	case pb.MessageType_SendBci_Request:
		bytes := message.GetMsgByte()
		request := new(pb.SendBciRequest)
		err := proto.Unmarshal(bytes, request)
		if err != nil {
			return err
		}
		_, err = e.Worker.handleSendBciRequest(request)
		e.log.Trace("Received SendBci request")
		if err != nil {
			return err
		}
	// case pb.MessageType_DevastateBci_Request:
	// 	bytes := message.GetMsgByte()
	// 	request := new(pb.DevastateBciRequest)
	// 	err := proto.Unmarshal(bytes, request)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	_, err = e.Worker.handleDevastateBciRequest(request)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	e.log.Error("[Engine]Get Devastate Bci request")
	case pb.MessageType_Client_Transaction:
		bytes := message.GetMsgByte()
		clienttrsaction := new(pb.ClientTransaction)
		err := proto.Unmarshal(bytes, clienttrsaction)
		if err != nil {
			return err
		}
		pbrawtx := clienttrsaction.GetTx()
		rawtx := types.ToRawTx(pbrawtx)
		txtype := clienttrsaction.GetTxType()
		switch txtype {
		case pb.TxType_CreateLockTransaction:
			err := e.Worker.checkLockTransaction(rawtx)
			if err != nil {
				return err
			}
			e.Worker.mempool.AddRawTx(rawtx)
		case pb.TxType_LockTransferTranscation:
			err := e.Worker.CheckLockTransferTransaction(rawtx)
			if err != nil {
				return err
			}
			e.Worker.mempool.AddRawTx(rawtx)
		case pb.TxType_NonLockTransferTranscation:
			err := e.Worker.CheckNonLockTransferTransaction(rawtx)
			if err != nil {
				return err
			}
			e.Worker.mempool.AddRawTx(rawtx)
		case pb.TxType_DevasteTransaction:
			err := e.Worker.CheckDevastateTransaction(rawtx)
			if err != nil {
				return err
			}
			e.Worker.mempool.AddRawTx(rawtx)
		}
	}
	return nil
}

func (e *PoTEngine) handleblock(b *types.Block) {
	if b != nil {
		channel := e.Worker.GetPeerQueue()
		channel <- b
	}

}

func DecodePacket(packet []byte) (*pb.Packet, error) {
	p := new(pb.Packet)
	err := proto.Unmarshal(packet, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
