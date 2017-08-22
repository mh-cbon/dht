// Package bucket implements the Kademlia binary tree.
package bucket

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"sort"
)

// New bucket with given id. Given id is preferably a 20 bytes string.
func New(targetID []byte) *KBucket {
	ret := &KBucket{}
	ret.Clear()
	ret.targetID = targetID
	return ret
}

// Clear the bucket
func (k *KBucket) Clear() {
	k.distance = Distance
	k.arbiter = Arbiter
	k.numberOfNodesPerKBucket = 20
	k.numberOfNodesToPing = 3
	k.root = createNode()
	if k.targetID != nil {
		k.targetID = k.targetID[:0]
	}
	k.ping = nil
	k.added = nil
	k.removed = nil
	k.updated = nil
	k.targetID = k.targetID[:0]
}

// Configure the bucket
func (k *KBucket) Configure(numberOfNodesPerKBucket, numberOfNodesToPing int) {
	k.numberOfNodesPerKBucket = numberOfNodesPerKBucket
	k.numberOfNodesToPing = numberOfNodesToPing
}

// ID returns a [20]byte
func (k *KBucket) ID() (ret [20]byte) {
	for i, b := range k.targetID {
		if i < 20 {
			ret[i] = b
		}
	}
	return
}

// OnPing registers the ping function responsible to re-add old contacts provided.
func (k *KBucket) OnPing(p PingFunc) {
	k.ping = p
}

// OnAdded registers a function called when a contact is added.
// Excluding duplicated node.
// It is also called when a node cant be split.
// todo: check doc.
func (k *KBucket) OnAdded(p AddFunc) {
	k.added = p
}

// OnRemoved is called when a contact is removed.
func (k *KBucket) OnRemoved(p RemoveFunc) {
	k.removed = p
}

// OnUpdated is called when a contact is updated.
func (k *KBucket) OnUpdated(p UpdateFunc) {
	k.updated = p
}

// KBucket is a Kademlia DHT K-bucket of nodes that registers contacts.
// It is a non-ts implementation.
type KBucket struct {
	root                    *bucketNode
	targetID                []byte
	distance                DistanceFunc
	arbiter                 ArbiterFunc
	numberOfNodesPerKBucket int
	numberOfNodesToPing     int
	ping                    PingFunc
	added                   AddFunc
	removed                 RemoveFunc
	updated                 UpdateFunc
}

// bucketNode isa list of contacts,
// which will split into left/right bucketNodes,
// when its contacts list is full.
type bucketNode struct {
	contacts  []ContactIdentifier
	dontSplit bool
	left      *bucketNode
	right     *bucketNode
}

// append given contact.
func (n *bucketNode) append(contact ContactIdentifier) {
	n.contacts = append(n.contacts, contact)
}

// search for given contact.
func (n *bucketNode) indexOf(id []byte) int {
	for i, c := range n.contacts {
		if bytes.Equal(c.GetID(), id) {
			return i
		}
	}
	return -1
}

// remove a contact at given index.
func (n *bucketNode) removeAt(index int) ContactIdentifier {
	if index > len(n.contacts) {
		return nil
	}
	var contact ContactIdentifier
	o := n.contacts[0:]
	contact = o[index]
	n.contacts = n.contacts[:0]
	for i := range o {
		if i != index {
			n.contacts = append(n.contacts, o[i])
		}
	}
	return contact
}

// return a contact at given index
func (n *bucketNode) atIndex(index int) ContactIdentifier {
	if index >= len(n.contacts) {
		return nil
	}
	return n.contacts[index]
}

// count the number of contacts.
func (n *bucketNode) count() int {
	return len(n.contacts)
}

// RemoveByAddr a contact by its address.
// address: Address of the contact to remove.
func (n *bucketNode) removeByAddr(addr *net.UDPAddr) (ret []ContactIdentifier) {
	nContacts := []ContactIdentifier{}
	for _, c := range n.contacts {
		if c.GetAddr().String() == addr.String() {
			ret = append(ret, c)
		} else {
			nContacts = append(nContacts, c)
		}
	}
	n.contacts = nContacts
	return ret
}

// NewContact creates a new KContact
func NewContact(id string, addr net.UDPAddr) *KContact {
	return &KContact{[]byte(id), &addr, 0}
}

// ContactIdentifier is a contact with an id and an address
type ContactIdentifier interface {
	GetAddr() *net.UDPAddr
	GetID() []byte
}

// ContactVectorProvider is a contact with a vector to identify newest instance between two contacts with the same id
type ContactVectorProvider interface {
	GetVectorClock() int
}

// KContact has an id, an address, and a vectorclock.
type KContact struct {
	id          []byte
	addr        *net.UDPAddr
	vectorClock int // not sure how to use this.
}

// GetAddr ...
func (c KContact) GetAddr() *net.UDPAddr {
	return c.addr
}

// GetID ...
func (c KContact) GetID() []byte {
	return c.id
}

// SetVectorClock ...
func (c *KContact) SetVectorClock(v int) { c.vectorClock = v }

// GetVectorClock ...
func (c *KContact) GetVectorClock() int { return c.vectorClock }

func (c *KContact) idEq(other []byte) bool {
	return bytes.Equal(c.id[:], other[:])
}

func createNode() *bucketNode {
	return &bucketNode{contacts: []ContactIdentifier{}}
}

// PingFunc should ping each oldContacts,
// if it responds it should be re-added (k.Add).
// This puts the old contact on the "recently heard from" end of the list of nodes in the k-bucket.
// If the old contact does not respond, it should be removed (K.remove)
// and the new contact being added now has room to be stored (K.add).
type PingFunc func(oldContacts []ContactIdentifier, newishContact ContactIdentifier)

// AddFunc is invoked when a contact is added.
type AddFunc func(addedContact ContactIdentifier)

// RemoveFunc is called when a contact is removed.
type RemoveFunc func(removedContact ContactIdentifier)

// UpdateFunc is called when a contact is updated (vetcorClock).
type UpdateFunc func(oldContact ContactIdentifier, newContact ContactIdentifier)

// DistanceFunc computes the distance between to contact IDs.
type DistanceFunc func(firstId, secondId []byte) float64

// ArbiterFunc resolves conflicts
// when two node with identical ID and perhaps different properties,
// are added in the store.
type ArbiterFunc func(incumbent, candidate ContactVectorProvider) ContactIdentifier

// Arbiter resolves conflicts for two *KContacts.
func Arbiter(incumbent, candidate ContactVectorProvider) ContactIdentifier {
	if incumbent.GetVectorClock() > candidate.GetVectorClock() {
		return incumbent.(ContactIdentifier)
	}
	return candidate.(ContactIdentifier)
}

// Distance computes the Kadmelia distance.
func Distance(firstID, secondID []byte) float64 {
	var distance float64
	min := math.Min(float64(len(firstID)), float64(len(secondID)))
	max := math.Max(float64(len(firstID)), float64(len(secondID)))
	var i int
	for i = 0; i < int(min); i++ {
		distance = distance*256 + float64((firstID[i] ^ secondID[i]))
	}
	for ; i < int(max); i++ {
		distance = distance*256 + 255
	}
	return distance
}

// AddMany contacts to the bucket. Stop on first error.
func (k *KBucket) AddMany(contacts ...*KContact) error {
	for _, contact := range contacts {
		if err := k.Add(contact); err != nil {
			return err
		}
	}
	return nil
}

// Add the contact
func (k *KBucket) Add(contact ContactIdentifier) error {
	if contact == nil {
		return fmt.Errorf("contact must not be nil")
	}
	if len(contact.GetID()) > 20 {
		return fmt.Errorf("contact id len must lteq 20")
	}

	bitIndex := 0
	node := k.root
	for node.contacts == nil {
		// this is not a leaf node but an inner node with 'low' and 'high'
		// branches; we will check the appropriate bit of the identifier and
		// delegate to the appropriate node for further processing
		node = k.determineNode(node, contact.GetID(), bitIndex)
		bitIndex++
	}

	// check if the contact already exists
	index := node.indexOf(contact.GetID())
	if index > -1 {
		k.update(node, index, contact)
		return nil
	}

	if node.count() < k.numberOfNodesPerKBucket {
		node.append(contact)
		if k.added != nil {
			go k.added(contact)
		}
		return nil
	}

	// the bucket is full
	if node.dontSplit {
		// we are not allowed to split the bucket
		// we need to ping the first this.numberOfNodesToPing
		// in order to determine if they are alive
		// only if one of the pinged nodes does not respond, can the new contact
		// be added (this prevents DoS flodding with new invalid contacts)
		if k.ping != nil {
			contacts := []ContactIdentifier{}
			for _, c := range node.contacts[:k.numberOfNodesToPing] {
				contacts = append(contacts, c)
			}
			go k.ping(contacts, contact)
		}
		return nil
	}
	k.split(node, bitIndex)
	return k.Add(contact)
}

type contactsSorter []ContactIdentifier

type distContactsSorter struct {
	contactsSorter
	id       []byte
	distance DistanceFunc
}

func (k *distContactsSorter) Len() int { return len(k.contactsSorter) }
func (k *distContactsSorter) Swap(i, j int) {
	c := k.contactsSorter
	c[i], c[j] = c[j], c[i]
}
func (k *distContactsSorter) Less(i, j int) bool {
	c := k.contactsSorter
	delta := k.distance(c[i].GetID(), k.id) - k.distance(k.id, c[j].GetID())
	return delta < 0
}

// Closest contacts for given id.
// id: Buffer *required* node id
// n: Integer (Default: Infinity) maximum number of closest contacts to return
// Return: Array of maximum of `n` closest contacts to the node id
func (k *KBucket) Closest(id []byte, n uint) []ContactIdentifier {
	ret := []ContactIdentifier{}

	bitIndex := 0
	var nodes []*bucketNode
	nodes = append(nodes, k.root)
	for len(nodes) > 0 && len(ret) < int(n) {
		node := nodes[len(nodes)-1:][0]
		nodes = nodes[:len(nodes)-1]
		if node.contacts == nil {
			detNode := k.determineNode(node, id, bitIndex)
			if detNode == node.left {
				if node.right != nil {
					nodes = append(nodes, node.right)
				}
			} else {
				if node.left != nil {
					nodes = append(nodes, node.left)
				}
			}
			if detNode != nil {
				nodes = append(nodes, detNode)
			}
			bitIndex++
		} else {
			y := make([]ContactIdentifier, len(node.contacts))
			copy(y, node.contacts)
			sort.Sort(&distContactsSorter{contactsSorter(y), id, k.distance})
			ret = append(ret, y...)
			if len(ret) > int(n) {
				ret = ret[:n]
			}
		}
	}

	return ret
}

// Count the number of contacts recursively.
// If this is a leaf, just return the number of contacts contained. Otherwise,
// return the length of the high and low branches combined.
func (k *KBucket) Count() (count int) {
	k.visit(func(node *bucketNode) {
		count += node.count()
	})
	return
}

// IndexOf Returns the index of the contact with the given id if it exists
// node: internal object that has 2 leafs: left and right
// id: Buffer Contact node id.
func (k *KBucket) IndexOf(node *bucketNode, id []byte) int {
	return node.indexOf(id)
}

// determineNode Determines whether the id at the bitIndex is 0 or 1.
// Return left leaf if `id` at `bitIndex` is 0, right leaf otherwise
// node: internal object that has 2 leafs: left and right
// id: a Buffer to compare localNodeId with
// bitIndex: the bitIndex to which bit to check in the id Buffer
func (k *KBucket) determineNode(node *bucketNode, id []byte, bitIndex int) *bucketNode {
	if len(id) < 20 {
		panic(fmt.Errorf("id length must be 20, got %v", id))
	}
	// **NOTE** remember that id is a Buffer and has granularity of
	// bytes (8 bits), whereas the bitIndex is the _bit_ index (not byte)

	// id's that are too short are put in low bucket (1 byte = 8 bits)
	// parseInt(bitIndex / 8) finds how many bytes the bitIndex describes
	// bitIndex % 8 checks if we have extra bits beyond byte multiples
	// if number of bytes is <= no. of bytes described by bitIndex and there
	// are extra bits to consider, this means id has less bits than what
	// bitIndex describes, id therefore is too short, and will be put in low
	// bucket
	bytesDescribedByBitIndex := int(math.Floor(float64(bitIndex) / 8))
	bitIndexWithinByte := float64(bitIndex % 8)

	if len(id) <= bytesDescribedByBitIndex && bitIndexWithinByte != 0 {
		return node.left
	}

	byteUnderConsideration := id[bytesDescribedByBitIndex]

	// byteUnderConsideration is an integer from 0 to 255 represented by 8 bits
	// where 255 is 11111111 and 0 is 00000000
	// in order to find out whether the bit at bitIndexWithinByte is set
	// we construct Math.pow(2, (7 - bitIndexWithinByte)) which will consist
	// of all bits being 0, with only one bit set to 1
	// for example, if bitIndexWithinByte is 3, we will construct 00010000 by
	// Math.pow(2, (7 - 3)) -> Math.pow(2, 4) -> 16
	y := int(byteUnderConsideration) & int(math.Pow(2, (7-bitIndexWithinByte)))

	if y > 0 {
		return node.right
	}

	return node.left
}

// Get a contact by its exact ID.
// If this is a leaf, loop through the bucket contents and return the correct
// contact if we have it or null if not. If this is an inner node, determine
// which branch of the tree to traverse and repeat.
// id: Buffer *required* The ID of the contact to fetch.
func (k *KBucket) Get(id []byte) ContactIdentifier {

	bitIndex := 0
	node := k.root
	for node.contacts == nil {
		if n := k.determineNode(node, id, bitIndex); n != nil {
			node = n
		} else {
			break
		}
		bitIndex++
	}

	if node == nil {
		return nil
	}

	index := node.indexOf(id) // index of uses contact id for matching
	if index > -1 {
		return node.atIndex(index)
	}
	return nil
}

// RemoveByAddr a contact by its address.
// address: Address of the contact to remove.
func (k *KBucket) RemoveByAddr(addr *net.UDPAddr) (ret []ContactIdentifier) {

	bitIndex := 0
	var nodes []*bucketNode
	nodes = append(nodes, k.root)
	for len(nodes) > 0 {
		node := nodes[len(nodes)-1:][0]
		nodes = nodes[:len(nodes)-1]
		if node.contacts == nil {
			if node.right != nil {
				nodes = append(nodes, node.right)
			}
			if node.left != nil {
				nodes = append(nodes, node.left)
			}
			bitIndex++
		} else {
			ret = append(ret, node.removeByAddr(addr)...)
		}
	}

	return ret
}

// Remove a contact by its ID.
// id: Buffer *required* The ID of the contact to remove.
func (k *KBucket) Remove(id []byte) ContactIdentifier {

	bitIndex := 0
	node := k.root
	for node.contacts == nil {
		if n := k.determineNode(node, id, bitIndex); n != nil {
			node = n
		} else {
			break
		}
		bitIndex++
	}

	index := node.indexOf(id)
	if index > -1 {
		contact := node.removeAt(index)
		if k.removed != nil && contact != nil {
			go k.removed(contact)
		}
		return contact
	}
	return nil
}

// Splits the node, redistributes contacts to the new nodes, and marks the
// node that was split as an inner node of the binary tree of nodes by
// setting this.root.contacts = null
// node: *required* node for splitting
// bitIndex: *required* the bitIndex to which byte to check in the Buffer
//          for navigating the binary tree
func (k *KBucket) split(node *bucketNode, bitIndex int) {
	node.left = createNode()
	node.right = createNode()

	// redistribute existing contacts amongst the two newly created nodes
	for i := 0; i < node.count(); i++ {
		contact := node.atIndex(i)
		detNode := k.determineNode(node, contact.GetID(), bitIndex)
		if detNode != nil {
			detNode.append(contact)
		}
	}
	node.contacts = nil // mark as inner tree node

	// don't split the "far away" node
	// we check where the local node would end up and mark the other one as
	// "dontSplit" (i.e. "far away")
	detNode := k.determineNode(node, k.targetID, bitIndex)
	otherNode := node.left
	if node.left == detNode {
		otherNode = node.right
	}
	otherNode.dontSplit = true
}

// ToArray Returns all the contacts contained in the tree as an array.
// If this is a leaf, return a copy of the bucket. `slice` is used so that we
// don't accidentally leak an internal reference out that might be accidentally
// misused. If this is not a leaf, return the union of the low and high
// branches (themselves also as arrays).
func (k *KBucket) ToArray() (ret []ContactIdentifier) {
	k.visit(func(node *bucketNode) {
		ret = append(ret, node.contacts...)
	})
	return ret
}

// recursive traversal of bucketNodes
func (k *KBucket) visit(visiter func(*bucketNode)) {
	var nodes []*bucketNode
	nodes = append(nodes, k.root)
	for len(nodes) > 0 {
		node := nodes[len(nodes)-1:][0]
		nodes = nodes[:len(nodes)-1]
		if node.contacts == nil {
			if node.right != nil {
				nodes = append(nodes, node.right)
			}
			if node.left != nil {
				nodes = append(nodes, node.left)
			}
		} else {
			if visiter != nil {
				visiter(node)
			}
		}
	}
}

// Updates the contact selected by the arbiter.
// If the selection is our old contact and the candidate is some new contact
// then the new contact is abandoned (not added).
// If the selection is our old contact and the candidate is our old contact
// then we are refreshing the contact and it is marked as most recently
// contacted (by being moved to the right/end of the bucket array).
// If the selection is our new contact, the old contact is removed and the new
// contact is marked as most recently contacted.
// node: internal object that has 2 leafs: left and right
// contact: *required* the contact to update
// index: *required* the index in the bucket where contact exists
//        (index has already been computed in a previous calculation)
func (k *KBucket) update(node *bucketNode, index int, contact ContactIdentifier) error {
	// sanity check
	incumbent := node.atIndex(index)
	if bytes.Equal(incumbent.GetID(), contact.GetID()) == false {
		return fmt.Errorf("Id s mismatch %v %v", incumbent.GetID(), contact.GetID())
	}
	var vIncumbent ContactVectorProvider
	var vContact ContactVectorProvider
	var ok bool
	if vIncumbent, ok = incumbent.(ContactVectorProvider); ok == false {
		return fmt.Errorf("incumbent is not a ContactVectorProvider")
	}
	if vContact, ok = contact.(ContactVectorProvider); ok == false {
		return fmt.Errorf("incumbent is not a ContactVectorProvider")
	}

	selection := k.arbiter(vIncumbent, vContact)
	// if the selection is our old contact and the candidate is some new
	// contact, then there is nothing to do
	if selection == incumbent && incumbent != contact {
		return nil
	}

	node.removeAt(index)   // remove old contact
	node.append(selection) // add more recent contact version
	if k.updated != nil {
		go k.updated(incumbent, selection)
	}
	return nil
}
