package p2p

import (
	"context"
	"errors"
	"fmt"

	"github.com/NethermindEth/juno/adapters/p2p2core"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/db"
	"github.com/NethermindEth/juno/p2p/starknet"
	"github.com/NethermindEth/juno/p2p/starknet/spec"
	"github.com/NethermindEth/juno/utils"
)

func (s *syncService) startPipeline(ctx context.Context) {
	s.client = starknet.NewClient(s.randomPeerStream, s.network, s.log)

	bootNodeHeight, err := s.bootNodeHeight(ctx)
	if err != nil {
		s.log.Errorw("Failed to get boot node height", "err", err)
		return
	}
	s.log.Infow("Boot node height", "height", bootNodeHeight)

	var nextHeight uint64
	if curHeight, err := s.blockchain.Height(); err == nil { //nolint:govet
		nextHeight = curHeight + 1
	} else if !errors.Is(db.ErrKeyNotFound, err) {
		s.log.Errorw("Failed to get current height", "err", err)
	}

	commonIt := s.createIterator(BlockRange{nextHeight, bootNodeHeight})
	headersAndSigsCh, err := s.genHeadersAndSigs(ctx, commonIt)
	if err != nil {
		s.log.Errorw("Failed to get block headers parts", "err", err)
		return
	}

	blockBodiesCh, err := s.genBlockBodies(ctx, commonIt)
	if err != nil {
		s.log.Errorw("Failed to get block bodies", "err", err)
		return
	}

	txsCh, err := s.genTransactions(ctx, commonIt)
	if err != nil {
		s.log.Errorw("Failed to get transactions", "err", err)
		return
	}

	receiptsCh, err := s.genReceipts(ctx, commonIt)
	if err != nil {
		s.log.Errorw("Failed to get receipts", "err", err)
		return
	}

	eventsCh, err := s.genEvents(ctx, commonIt)
	if err != nil {
		s.log.Errorw("Failed to get events", "err", err)
		return
	}

	// A channel of a specific type cannot be converted to a channel of another type. Therefore, we have to consume/read from the channel
	// and change the input to the desired type. The following is not allowed:
	// var ch1 chan any = make(chan any)
	// var ch2 chan someOtherType = make(chan someOtherType)
	// ch2 = (chan any)(ch2) <----- This line will give compilation error.

	for block := range utils.PipelineBridge(ctx,
		processSpecBlockParts(ctx, nextHeight,
			utils.PipelineFanIn(ctx,
				utils.PipelineStage(ctx, headersAndSigsCh, func(i specBlockHeaderAndSigs) specBlockParts { return i }),
				utils.PipelineStage(ctx, blockBodiesCh, func(i specBlockBody) specBlockParts { return i }),
				utils.PipelineStage(ctx, txsCh, func(i specTransactions) specBlockParts { return i }),
				utils.PipelineStage(ctx, receiptsCh, func(i specReceipts) specBlockParts { return i }),
				utils.PipelineStage(ctx, eventsCh, func(i specEvents) specBlockParts { return i }),
			))) {
		fmt.Println("----- Got core block number:", block.block.Number, "-----")
	}
}

func processSpecBlockParts(ctx context.Context, startingBlockNum uint64, specBlockPartsCh <-chan specBlockParts) <-chan <-chan blockBody {
	orderedBlockBodiesCh := make(chan (<-chan blockBody))

	go func() {
		defer close(orderedBlockBodiesCh)

		specBlockHeaderAndSigsM := make(map[uint64]specBlockHeaderAndSigs)
		specBlockBodyM := make(map[uint64]specBlockBody)
		specTransactionsM := make(map[uint64]specTransactions)
		specReceiptsM := make(map[uint64]specReceipts)
		specEventsM := make(map[uint64]specEvents)

		curBlockNum := startingBlockNum
		for part := range specBlockPartsCh {
			select {
			case <-ctx.Done():
			default:
				switch p := part.(type) {
				case specBlockHeaderAndSigs:
					fmt.Printf("Received Block Header and Signatures for block number: %v\n", p.header.Number)
					if _, ok := specBlockHeaderAndSigsM[part.blockNumber()]; !ok {
						specBlockHeaderAndSigsM[part.blockNumber()] = p
					}
				case specBlockBody:
					fmt.Printf("Received Block Body parts            for block number: %v\n", p.id.Number)
					if _, ok := specBlockBodyM[part.blockNumber()]; !ok {
						specBlockBodyM[part.blockNumber()] = p
					}
				case specTransactions:
					fmt.Printf("Received Transactions                for block number: %v\n", p.id.Number)
					if _, ok := specTransactionsM[part.blockNumber()]; !ok {
						specTransactionsM[part.blockNumber()] = p
					}
				case specReceipts:
					fmt.Printf("Received Receipts                    for block number: %v\n", p.id.Number)
					if _, ok := specReceiptsM[part.blockNumber()]; !ok {
						specReceiptsM[part.blockNumber()] = p
					}
				case specEvents:
					fmt.Printf("Received Events                      for block number: %v\n", p.id.Number)
					if _, ok := specEventsM[part.blockNumber()]; !ok {
						specEventsM[part.blockNumber()] = p
					}
				}

				headerAndSig, ok1 := specBlockHeaderAndSigsM[curBlockNum]
				body, ok2 := specBlockBodyM[curBlockNum]
				txs, ok3 := specTransactionsM[curBlockNum]
				rs, ok4 := specReceiptsM[curBlockNum]
				es, ok5 := specEventsM[curBlockNum]
				if ok1 && ok2 && ok3 && ok4 && ok5 {
					fmt.Println("----- Received all block parts from peers for block number:", curBlockNum, "-----")

					select {
					case <-ctx.Done():
					case orderedBlockBodiesCh <- adaptAndSanityCheckBlock(ctx, headerAndSig.header, headerAndSig.sig, body.stateDiff,
						body.classes, body.proof, txs.txs, rs.receipts, es.events):
					}

					delete(specBlockHeaderAndSigsM, curBlockNum)
					delete(specBlockBodyM, curBlockNum)
					delete(specTransactionsM, curBlockNum)
					delete(specReceiptsM, curBlockNum)
					delete(specEventsM, curBlockNum)
					curBlockNum++
				}
			}
		}
	}()
	return orderedBlockBodiesCh
}

func adaptAndSanityCheckBlock(ctx context.Context, header *spec.BlockHeader, sig *spec.Signatures, diff *spec.StateDiff,
	classes *spec.Classes, proof *spec.BlockProof, txs *spec.Transactions, receipts *spec.Receipts, events *spec.Events,
) <-chan blockBody {
	bodyCh := make(chan blockBody)
	go func() {
		defer close(bodyCh)
		select {
		case <-ctx.Done():
			bodyCh <- blockBody{err: ctx.Err()}
		default:
			// Todo Process block later
			bodyCh <- blockBody{block: &core.Block{Header: &core.Header{Number: header.Number}}}
		}
	}()

	return bodyCh
}

type specBlockParts interface {
	blockNumber() uint64
}

type specBlockHeaderAndSigs struct {
	header *spec.BlockHeader
	sig    *spec.Signatures
}

func (s specBlockHeaderAndSigs) blockNumber() uint64 {
	return s.header.Number
}

func (s *syncService) genHeadersAndSigs(ctx context.Context, it *spec.Iteration) (<-chan specBlockHeaderAndSigs, error) {
	headersIt, err := s.client.RequestBlockHeaders(ctx, &spec.BlockHeadersRequest{Iteration: it})
	if err != nil {
		return nil, err
	}

	headersAndSigCh := make(chan specBlockHeaderAndSigs)
	go func() {
		defer close(headersAndSigCh)
		for res, valid := headersIt(); valid; res, valid = headersIt() {
			headerAndSig := specBlockHeaderAndSigs{}
			for _, part := range res.GetPart() {
				switch part.HeaderMessage.(type) {
				case *spec.BlockHeadersResponsePart_Header:
					headerAndSig.header = part.GetHeader()
				case *spec.BlockHeadersResponsePart_Signatures:
					headerAndSig.sig = part.GetSignatures()
				case *spec.BlockHeadersResponsePart_Fin:
					// received all the parts of BlockHeadersResponse
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case headersAndSigCh <- headerAndSig:
			}
		}
	}()

	return headersAndSigCh, nil
}

func adaptBlockHeadersAndSigs(headerAndSig specBlockHeaderAndSigs) core.Header {
	header := p2p2core.AdaptBlockHeader(headerAndSig.header)
	header.Signatures = utils.Map(headerAndSig.sig.GetSignatures(), p2p2core.AdaptSignature)
	return header
}

type specBlockBody struct {
	id        *spec.BlockID
	proof     *spec.BlockProof
	classes   *spec.Classes
	stateDiff *spec.StateDiff
}

func (s specBlockBody) blockNumber() uint64 {
	return s.id.Number
}

func (s *syncService) genBlockBodies(ctx context.Context, it *spec.Iteration) (<-chan specBlockBody, error) {
	blockIt, err := s.client.RequestBlockBodies(ctx, &spec.BlockBodiesRequest{Iteration: it})
	if err != nil {
		return nil, err
	}

	specBodiesCh := make(chan specBlockBody)
	go func() {
		defer close(specBodiesCh)
		curBlockBody := new(specBlockBody)
		// Assumes that all parts of the same block will arrive before the next block parts
		// Todo: the above assumption may not be true. A peer may decide to send different parts of the block in different order
		// If the above assumption is not true we should return separate channels for each of the parts. Also, see todo above specBlockBody
		// on line 317 in p2p/sync.go
		for res, valid := blockIt(); valid; res, valid = blockIt() {
			switch res.BodyMessage.(type) {
			case *spec.BlockBodiesResponse_Classes:
				if curBlockBody.id == nil {
					curBlockBody.id = res.GetId()
				}
				curBlockBody.classes = res.GetClasses()
			case *spec.BlockBodiesResponse_Diff:
				if curBlockBody.id == nil {
					curBlockBody.id = res.GetId()
				}
				curBlockBody.stateDiff = res.GetDiff()
			case *spec.BlockBodiesResponse_Proof:
				if curBlockBody.id == nil {
					curBlockBody.id = res.GetId()
				}
				curBlockBody.proof = res.GetProof()
			case *spec.BlockBodiesResponse_Fin:
				if curBlockBody.id != nil {
					select {
					case <-ctx.Done():
					default:
						specBodiesCh <- *curBlockBody
						curBlockBody = new(specBlockBody)
					}
				}
			}
		}
	}()

	return specBodiesCh, nil
}

type specReceipts struct {
	id       *spec.BlockID
	receipts *spec.Receipts
}

func (s specReceipts) blockNumber() uint64 {
	return s.id.Number
}

func (s *syncService) genReceipts(ctx context.Context, it *spec.Iteration) (<-chan specReceipts, error) {
	receiptsIt, err := s.client.RequestReceipts(ctx, &spec.ReceiptsRequest{Iteration: it})
	if err != nil {
		return nil, err
	}

	receiptsCh := make(chan specReceipts)
	go func() {
		defer close(receiptsCh)

		for res, valid := receiptsIt(); valid; res, valid = receiptsIt() {
			switch res.Responses.(type) {
			case *spec.ReceiptsResponse_Receipts:
				select {
				case <-ctx.Done():
				case receiptsCh <- specReceipts{res.GetId(), res.GetReceipts()}:
				}
			case *spec.ReceiptsResponse_Fin:
				return
			}
		}
	}()

	return receiptsCh, nil
}

type specEvents struct {
	id     *spec.BlockID
	events *spec.Events
}

func (s specEvents) blockNumber() uint64 {
	return s.id.Number
}

func (s *syncService) genEvents(ctx context.Context, it *spec.Iteration) (<-chan specEvents, error) {
	eventsIt, err := s.client.RequestEvents(ctx, &spec.EventsRequest{Iteration: it})
	if err != nil {
		return nil, err
	}

	eventsCh := make(chan specEvents)
	go func() {
		defer close(eventsCh)
		for res, valid := eventsIt(); valid; res, valid = eventsIt() {
			switch res.Responses.(type) {
			case *spec.EventsResponse_Events:
				select {
				case <-ctx.Done():
				case eventsCh <- specEvents{res.GetId(), res.GetEvents()}:
				}
			case *spec.EventsResponse_Fin:
				return
			}
		}
	}()
	return eventsCh, nil
}

type specTransactions struct {
	id  *spec.BlockID
	txs *spec.Transactions
}

func (s specTransactions) blockNumber() uint64 {
	return s.id.Number
}

func (s *syncService) genTransactions(ctx context.Context, it *spec.Iteration) (<-chan specTransactions, error) {
	txsIt, err := s.client.RequestTransactions(ctx, &spec.TransactionsRequest{Iteration: it})
	if err != nil {
		return nil, err
	}

	txsCh := make(chan specTransactions)
	go func() {
		defer close(txsCh)
		for res, valid := txsIt(); valid; res, valid = txsIt() {
			switch res.Responses.(type) {
			case *spec.TransactionsResponse_Transactions:
				select {
				case <-ctx.Done():
				case txsCh <- specTransactions{res.GetId(), res.GetTransactions()}:
				}
			case *spec.TransactionsResponse_Fin:
				return
			}
		}
	}()
	return txsCh, nil
}
