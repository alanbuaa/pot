package pot

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
	"time"
)

func (w *Worker) request(request *pb.HeaderRequest) (*pb.HeaderResponse, error) {
	requestByte, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	potMsg := &pb.PoTMessage{
		MsgType: pb.MessageType_Header_Request,
		MsgByte: requestByte,
	}
	potMsgByte, err := proto.Marshal(potMsg)
	if err != nil {
		return nil, err
	}

	for {
		if w.headerResponseChan == nil {
			ch := make(chan *pb.HeaderResponse, 10)
			w.headerResponseChan = ch
			break
		}
		time.Sleep(1 * time.Second)
	}

	err = w.Engine.Unicast(request.Des, potMsgByte)

	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(5 * time.Second)

	res := new(pb.HeaderResponse)
	select {
	case response := <-w.headerResponseChan:
		if response.Src != request.Des || response.Srcid != request.Desid {
			return nil, fmt.Errorf("receive response from wrong address")
		}
		res = response
		w.log.Infof("[PoT]\treceive for header %s from %s", hexutil.Encode(request.GetHashes()), request.GetDes())
		timer.Stop()
	case <-timer.C:

		close(w.headerResponseChan)
		w.headerResponseChan = nil
		timer.Stop()
		return nil, fmt.Errorf("request for header %s from %s timeout", hexutil.Encode(request.GetHashes()), request.GetDes())
	}

	close(w.headerResponseChan)
	w.headerResponseChan = nil
	if res != nil {
		return res, nil
	} else {
		return nil, fmt.Errorf("didn't receive response")
	}

}

func (w *Worker) handleHeaderResponse(response *pb.HeaderResponse) error {
	if w.headerResponseChan == nil {
		return fmt.Errorf("can't find channel to handle header response")
	} else {
		w.headerResponseChan <- response
		return nil
	}
}

func (w *Worker) handlePoTResponse(response *pb.PoTResponse) error {
	if w.potResponseCh == nil {
		return fmt.Errorf("can't find channel to handle pot response")
	} else {
		w.potResponseCh <- response
		return nil
	}
}

func (w *Worker) requestPoTResFor(epoch uint64, address int64, peerid string) ([]byte, error) {
	request := &pb.PoTRequest{
		Epoch: epoch,
		Desid: address,
		Des:   peerid,
		Srcid: w.ID,
		Src:   w.PeerId,
	}
	response, err := w.potRequest(request)
	if err != nil {
		return nil, err
	}
	res := response.GetProof()
	return res, nil

}

func (w *Worker) potRequest(request *pb.PoTRequest) (*pb.PoTResponse, error) {
	requestByte, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	potMsg := &pb.PoTMessage{
		MsgType: pb.MessageType_PoT_Request,
		MsgByte: requestByte,
	}
	msgByte, err := proto.Marshal(potMsg)
	if err != nil {
		return nil, err
	}
	counter := 0
	for {
		if w.potResponseCh == nil {
			ch := make(chan *pb.PoTResponse, 10)
			w.potResponseCh = ch
			break
		}

		time.Sleep(1 * time.Second)
		counter += 1
	}
	err = w.Engine.Unicast(request.Des, msgByte)
	if err != nil {
		return nil, err
	}
	res := new(pb.PoTResponse)
	select {
	case response := <-w.potResponseCh:
		if response.Src != request.Des || response.Srcid != request.Desid {
			return nil, fmt.Errorf("receive response from wrong address")
		}
		res = response
	}
	close(w.potResponseCh)
	w.potResponseCh = nil
	if res != nil {
		return res, nil
	} else {
		return nil, fmt.Errorf("didn't receive response")
	}
	// return nil, nil
}
func (w *Worker) getParentBlock(header *types.Header) (*types.Header, error) {
	if header == nil {
		return nil, fmt.Errorf("could not get parent from a nil block")
	}
	if header.Height == 1 {
		return types.DefaultGenesisHeader(), nil
	}
	parentHash := header.ParentHash

	if parentHash == nil {
		return nil, fmt.Errorf("the epcoh %d block %s from %d without parent", header.Height, hexutil.Encode(header.Hashes), header.Address)
	}
	parent, err := w.storage.Get(parentHash)

	if err != nil {
		request := &pb.HeaderRequest{
			Height: header.Height - 1,
			Hashes: parentHash,
			Desid:  header.Address,
			Srcid:  w.ID,
			Des:    header.PeerId,
			Src:    w.PeerId,
		}
		// w.p2p.Unicast()
		headerResponse, err := w.request(request)

		if err != nil {
			return nil, err
		}

		pbParent := headerResponse.GetHeader()
		parent = types.ToHeader(pbParent)

		flag, err := w.checkHeader(parent)

		if flag {
			w.storage.Put(parent)
			return parent, nil
		} else {
			return nil, err
		}
	} else {
		return parent, nil
	}
	// return nil, nil
}

func (w *Worker) getUncleBlock(header *types.Header) ([]*types.Header, error) {
	n := len(header.UncleHash)
	uncleHeaders := make([]*types.Header, n)

	for i := 0; i < n; i++ {
		uncleHeader, err := w.storage.Get(header.UncleHash[i])
		if err != nil {
			request := &pb.HeaderRequest{
				Height: header.Height - 1,
				Hashes: header.UncleHash[i],
				Desid:  header.Address,
				Srcid:  w.ID,
				Des:    header.PeerId,
				Src:    w.PeerId,
			}
			// w.p2p.Unicast()
			headerResponse, err := w.request(request)
			if err != nil {
				return nil, err
			}
			pbUncle := headerResponse.GetHeader()
			uncle := types.ToHeader(pbUncle)
			flag, err := w.checkHeader(uncle)
			if flag {
				w.storage.Put(uncle)
				uncleHeader = uncle
			} else {
				return nil, err
			}
		}
		uncleHeaders[i] = uncleHeader
	}
	return uncleHeaders, nil
}
