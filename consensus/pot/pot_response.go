package pot

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
	"time"
)

func (w *Worker) request(request *pb.BlockRequest) (*pb.BlockResponse, error) {
	requestByte, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	potMsg := &pb.PoTMessage{
		MsgType: pb.MessageType_Block_Request,
		MsgByte: requestByte,
	}
	potMsgByte, err := proto.Marshal(potMsg)
	if err != nil {
		return nil, err
	}

	for {
		if w.blockResponseChan == nil {
			ch := make(chan *pb.BlockResponse, 10)
			w.blockResponseChan = ch
			break
		}
		time.Sleep(1 * time.Second)
	}

	err = w.Engine.Unicast(request.Des, potMsgByte)

	if err != nil {
		return nil, err
	}
	duration := time.Duration(w.config.PoT.Timeout)
	timer := time.NewTimer(duration * time.Second)

	res := new(pb.BlockResponse)
	select {
	case response := <-w.blockResponseChan:
		if response.Src != request.Des {
			return nil, fmt.Errorf("receive response from wrong address")
		}
		res = response
		w.log.Infof("[PoT]\treceive for header %s from %s", hexutil.Encode(request.GetHashes()), request.GetDes())
		timer.Stop()
	case <-timer.C:

		close(w.blockResponseChan)
		w.blockResponseChan = nil
		timer.Stop()
		return nil, fmt.Errorf("request for header %s from %s timeout", hexutil.Encode(request.GetHashes()), request.GetDes())
	}

	close(w.blockResponseChan)
	w.blockResponseChan = nil
	if res != nil {
		return res, nil
	} else {
		return nil, fmt.Errorf("didn't receive response")
	}

}

func (w *Worker) handleBlockResponse(response *pb.BlockResponse) error {
	if w.blockResponseChan == nil {
		return fmt.Errorf("can't find channel to handle header response")
	} else {
		w.blockResponseChan <- response
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

		Des: peerid,

		Src: w.PeerId,
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
		if response.Src != request.Des {
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
func (w *Worker) getParentBlock(block *types.Block) (*types.Block, error) {
	if block == nil {
		return nil, fmt.Errorf("could not get parent from a nil block")
	}
	if block.GetHeader().Height == 1 {
		return types.DefaultGenesisBlock(), nil
	}

	//w.log.Warnf("request for parent block at height %d", block.GetHeader().Height)
	parentHash := block.GetHeader().ParentHash
	if parentHash == nil {
		return nil, fmt.Errorf("the epcoh %d block %s from %d without parent", block.GetHeader().Height, hexutil.Encode(block.Hash()), block.GetHeader().Address)
	}
	parent, err := w.blockStorage.Get(parentHash)

	if err != nil {
		request := &pb.BlockRequest{
			Height: block.GetHeader().Height - 1,
			Hashes: parentHash,

			Des: block.GetHeader().PeerId,
			Src: w.PeerId,
		}
		// w.p2p.Unicast()
		blockResponse, err := w.request(request)

		if err != nil {
			return nil, err
		}

		pbParent := blockResponse.GetBlock()
		parent = types.ToBlock(pbParent)

		flag, err := w.checkblock(parent)

		if flag {
			w.blockStorage.Put(parent)
			return parent, nil
		} else {
			return nil, err
		}
	} else {
		return parent, nil
	}
	// return nil, nil
}

func (w *Worker) getUncleBlock(block *types.Block) ([]*types.Block, error) {

	if block.GetHeader().Height == 1 || block.GetHeader().Height == 0 {
		return nil, nil
	}
	n := len(block.GetHeader().UncleHash)
	uncleblock := make([]*types.Block, n)
	header := block.GetHeader()
	for i := 0; i < n; i++ {
		uncleHeader, err := w.blockStorage.Get(header.UncleHash[i])
		if err != nil {
			request := &pb.BlockRequest{
				Height: header.Height - 1,
				Hashes: header.UncleHash[i],

				Des: header.PeerId,
				Src: w.PeerId,
			}
			// w.p2p.Unicast()
			blockResponse, err := w.request(request)
			if err != nil {
				return nil, err
			}
			pbUncle := blockResponse.GetBlock()
			uncle := types.ToBlock(pbUncle)
			flag, err := w.checkblock(uncle)
			if flag {
				w.blockStorage.Put(uncle)
				uncleHeader = uncle
			} else {
				return nil, err
			}
		}
		uncleblock[i] = uncleHeader
	}
	return uncleblock, nil
}
