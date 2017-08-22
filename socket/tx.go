package socket

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

//TxOpt is a option setter
type TxOpt func(*TxServer)

// TxOpts are TxServer options.
var TxOpts = struct {
	WithTimeout func(duraton time.Duration) TxOpt
}{
	WithTimeout: func(duration time.Duration) TxOpt {
		return func(s *TxServer) {
			s.queryTimeout = duration
		}
	},
}

// TxServer manages transactions.
type TxServer struct {
	transactions     map[txKey]*Tx
	transactionIDInt uint64
	mu               *sync.RWMutex
	stopped          bool
	queryTimeout     time.Duration
}

// NewTxServer creates a server of transactions.
func NewTxServer(opts ...TxOpt) *TxServer {
	ret := &TxServer{
		transactions: map[txKey]*Tx{},
		mu:           &sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

// Prepare creates a new transaction,
// invoke h with the new transaction,
// if h does not return an error, the transaction is saved.
func (s *TxServer) Prepare(addr *net.UDPAddr, onResponse func(kmsg.Msg), h func(*Tx) error) (*Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// tx := getTx()
	tx := newTx()

	tx.Setup(addr, s.nextTransactionID(), func(res kmsg.Msg) {
		// go func() {
		// 	tx.Reset()
		// 	putTx(tx)
		// }() //todo: fix transaction alloations.
		if onResponse != nil {
			onResponse(res)
		}
	})
	err := h(tx)
	if err != nil {
		return nil, err
	}
	err = s.addTransaction(tx)
	return tx, err
}

// HandleResponse looks for the transaction with given msg tx id,
// when found, it invokes its response handler,
// then deletes the transaction.
func (s *TxServer) HandleResponse(m kmsg.Msg, addr *net.UDPAddr) (*Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx := s.findResponseTransaction(m.T, addr)
	if tx == nil {
		return nil, fmt.Errorf("transaction id unknown: %q %q", addr, m.T)
	}
	go tx.handleResponse(m)
	s.deleteTransaction(tx)
	return tx, nil
}

// CancelAll transactions running, pending handlers won t be called.
func (s *TxServer) CancelAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.transactions {
		t.Cancel()
		s.deleteTransaction(t)
	}
	return nil
}

// Stop the server and cancel all requests.
func (s *TxServer) Stop() error {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	s.CancelAll()
	return nil
}

// Start loops over all transactions, if they timedout,
// invoke the response handler with a timeout error of code 201,
// then deletes the transaction.
func (s *TxServer) Start() {
	queryTimeout := s.queryTimeout
	if queryTimeout < 1 {
		queryTimeout = time.Second * 3
	}
	for {
		<-time.After(queryTimeout / 10)
		s.mu.Lock()
		for _, tx := range s.transactions {
			if s.stopped {
				s.mu.Unlock()
				return
			}
			if tx.hasTimeout(queryTimeout) {
				msg := kmsg.Msg{T: tx.txID, E: &kmsg.ErrorTimeout, Y: "q"}
				go tx.handleResponse(msg)
				s.deleteTransaction(tx)
			}
			if s.stopped {
				s.mu.Unlock()
				return
			}
		}
		s.mu.Unlock()
	}
}

func (s *TxServer) findResponseTransaction(transactionID string, sourceNode *net.UDPAddr) *Tx {
	return s.transactions[txKey{
		sourceNode.String(),
		transactionID}]
}

func (s *TxServer) nextTransactionID() string {
	var b [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(b[:], s.transactionIDInt)
	s.transactionIDInt++
	return string(b[:n])
}

func (s *TxServer) deleteTransaction(t *Tx) {
	delete(s.transactions, t.key())
}

func (s *TxServer) addTransaction(t *Tx) error {
	if _, ok := s.transactions[t.key()]; ok {
		return fmt.Errorf("transaction not unique %q %q", t.remote, t.txID)
	}
	s.transactions[t.key()] = t
	return nil
}

// Uniquely identifies a transaction to us.
type txKey struct {
	RemoteAddr string // host:port
	T          string // The KRPC transaction ID.
}

// Tx handles query/response sequence with timeout.
type Tx struct {
	txID    string
	remote  *net.UDPAddr
	started time.Time
	// query      map[string]interface{}
	onResponse func(kmsg.Msg)
	handled    func()
	mu         *sync.RWMutex
}

func newTx() *Tx {
	return (&Tx{mu: &sync.RWMutex{}})
}

// Setup the TX.
func (t *Tx) Setup(addr *net.UDPAddr, txID string, onResponse func(kmsg.Msg)) *Tx {
	t.onResponse = onResponse
	// t.query = query
	t.txID = txID
	t.remote = addr
	t.started = time.Now()
	return t
}

// Reset the TX.
func (t *Tx) Reset() {
	t.onResponse = nil
	// t.query = nil
	t.handled = nil
	t.txID = ""
	t.remote = nil
	t.started = time.Now()
}

func (t *Tx) key() txKey {
	return txKey{
		t.remote.String(),
		t.txID,
	}
}

// Cancel the transaction.
func (t *Tx) Cancel() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancel()
	return nil
}

func (t *Tx) cancel() error {
	if t.handled != nil {
		go t.handled()
	}
	t.onResponse = nil
	// t.query = nil
	return nil
}

func (t *Tx) hasTimeout(delta time.Duration) bool {
	return time.Now().After(t.started.Add(delta))
}

func (t *Tx) handleResponse(msg kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.onResponse != nil {
		t.onResponse(msg)
	}
	t.cancel()
}

func (t *Tx) onHandled(handled func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.onResponse == nil {
		// if t.onResponse == nil || t.query == nil {
		go handled()
	} else {
		t.handled = handled
	}
}

// var buffers = sync.Pool{
// 	New: func() interface{} {
// 		return newTx()
// 	},
// }
//
// func getTx() *Tx {
// 	return buffers.Get().(*Tx)
// }
//
// func putTx(tx *Tx) {
// 	buffers.Put(tx)
// }
