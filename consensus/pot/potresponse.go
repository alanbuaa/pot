package pot

import (
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

func (w *Worker) request(request *pb.HeaderRequest) (*pb.HeaderResponse, error) {
	requestbyte, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	potmsg := &pb.PoTMessage{
		MsgType: pb.MessageType_Header_Request,
		MsgByte: requestbyte,
	}
	msgbyte, err := proto.Marshal(potmsg)
	if err != nil {
		return nil, err
	}
	for {
		if w.headerResponsech == nil {
			ch := make(chan *pb.HeaderResponse, 10)
			w.headerResponsech = ch
			break
		}
		time.Sleep(1 * time.Second)
	}

	err = w.Engine.Unicast(request.Des, msgbyte)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(5 * time.Second)

	res := new(pb.HeaderResponse)
	select {
	case response := <-w.headerResponsech:
		if response.Src != request.Des || response.Srcid != request.Desid {
			return nil, fmt.Errorf("receive response from wrong address")
		}
		res = response
	case <-timer.C:
		close(w.headerResponsech)
		w.headerResponsech = nil
		return nil, fmt.Errorf("request for header %s from %s timeout", request.GetHashes(), request.GetDes())
	}

	close(w.headerResponsech)
	w.headerResponsech = nil
	if res != nil {
		return res, nil
	} else {
		return nil, fmt.Errorf("didn't receive response")
	}

}

func (w *Worker) handleHeaderResponse(response *pb.HeaderResponse) error {
	if w.headerResponsech == nil {
		return fmt.Errorf("can't find channel to handle header response")
	} else {
		w.headerResponsech <- response
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
		Src:   w.Peerid,
	}
	response, err := w.potRequest(request)
	if err != nil {
		return nil, err
	}
	res := response.GetProof()
	return res, nil

}

func (w *Worker) potRequest(request *pb.PoTRequest) (*pb.PoTResponse, error) {
	requestbyte, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	potmsg := &pb.PoTMessage{
		MsgType: pb.MessageType_PoT_Request,
		MsgByte: requestbyte,
	}
	msgbyte, err := proto.Marshal(potmsg)
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
	err = w.Engine.Unicast(request.Des, msgbyte)
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
