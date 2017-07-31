package bucket

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var localID = makeIDStr("local id")

func makeIDStr(in string) []byte {
	ret := []byte(in)
	if len(ret) < 20 {
		ret = append(ret, make([]byte, 20-len(ret))...)
	}
	return ret
}

func makeID(in ...byte) []byte {
	ret := make([]byte, 0)
	ret = append(ret, in...)
	for i := 0; i < 20-len(in); i++ {
		ret = append(ret, 0x00)
	}
	return ret
}

func TestNew(t *testing.T) {
	id := []byte{'a', 'a'}
	bucket := New(id)
	if len(bucket.ID()) != 20 {
		t.Error("Want bucket.localNodeId is []byte")
	}
	if bytes.Equal(bucket.targetID, id) == false {
		t.Errorf("Want bucket.localNodeId is %q, got %q", string(id), string(bucket.targetID))
	}
	if bucket.root == nil {
		t.Errorf("Want bucket.root is *bucketNode, got nil")
	}
	if bucket.root.dontSplit == true {
		t.Errorf("Want bucket.root.dontSplit is false")
	}
	if bucket.root.contacts == nil {
		t.Errorf("Want bucket.root.contacts is []*bucketNode, got nil")
	}
	if bucket.root.left != nil {
		t.Errorf("Want bucket.root.left is nil")
	}
	if bucket.root.right != nil {
		t.Errorf("Want bucket.root.right is nil")
	}
}

type distanceTest struct {
	from   []byte
	to     []byte
	expect float64
}

func TestDistance(t *testing.T) {
	testTable := []distanceTest{
		distanceTest{
			from:   []byte{0x00},
			to:     []byte{0x00},
			expect: 0,
		},
		distanceTest{
			from:   []byte{0x00},
			to:     []byte{0x01},
			expect: 1,
		},
		distanceTest{
			from:   []byte{0x02},
			to:     []byte{0x01},
			expect: 3,
		},
		distanceTest{
			from:   []byte{0x00},
			to:     []byte{0x00, 0x00},
			expect: 255,
		},
		distanceTest{
			from:   []byte{0x01, 0x24},
			to:     []byte{0x40, 0x24},
			expect: 16640,
		},
	}
	for _, data := range testTable {
		if got := Distance(data.from, data.to); got != data.expect {
			t.Errorf("Want Distance =%v, got = %v", data.expect, got)
		}
	}
}

type determineNodeTest struct {
	id       []byte
	wantNode *bucketNode
	bitIndex int
}

func TestDetermineNode(t *testing.T) {
	leftNode := &bucketNode{}
	rightNode := &bucketNode{}
	rootNode := &bucketNode{left: leftNode, right: rightNode}
	testTable := []determineNodeTest{
		determineNodeTest{
			id:       makeID(0x00),
			wantNode: leftNode,
			bitIndex: 0,
		},
		determineNodeTest{
			id:       makeID(0x40),
			wantNode: leftNode,
			bitIndex: 0,
		},
		determineNodeTest{
			id:       makeID(0x40),
			wantNode: rightNode,
			bitIndex: 1,
		},
		determineNodeTest{
			id:       makeID(0x40),
			wantNode: leftNode,
			bitIndex: 2,
		},
		determineNodeTest{
			id:       makeID(0x40),
			wantNode: leftNode,
			bitIndex: 9,
		},
		determineNodeTest{
			id:       makeID(0x41),
			wantNode: rightNode,
			bitIndex: 7,
		},
		determineNodeTest{
			id:       makeID(0x41, 0x00),
			wantNode: rightNode,
			bitIndex: 7,
		},
		determineNodeTest{
			id:       makeID(0x00, 0x41, 0x00),
			wantNode: rightNode,
			bitIndex: 15,
		},
	}
	for _, data := range testTable {
		bucket := New([]byte{})
		if got := bucket.determineNode(rootNode, data.id, data.bitIndex); got != data.wantNode {
			t.Errorf("Want determinNode = %v, got = %v", data.wantNode, got)
		}
	}
}

func TestAdd(t *testing.T) {

	t.Run("adding nil contact returns an error", func(t *testing.T) {
		bucket := New(localID)
		var contact ContactIdentifier
		wantErr := fmt.Errorf("contact must not be nil")
		gotErr := bucket.Add(contact)
		if gotErr.Error() != wantErr.Error() {
			t.Errorf("wanted err=%v got err =%v", wantErr, gotErr)
		}
	})

	t.Run("adding contact.id greater than 20 returns an error", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: []byte(strings.Repeat("foo", 20))}
		wantErr := fmt.Errorf("contact id len must lteq 20")
		gotErr := bucket.Add(contact)
		if gotErr.Error() != wantErr.Error() {
			t.Errorf("wanted err=%v got err =%v", wantErr, gotErr)
		}
	})

	t.Run("adding an existing contact does not increase number of contacts in root node", func(t *testing.T) {
		bucket := New(localID)
		contactID := makeIDStr("a")
		contact := &KContact{id: contactID}
		var wantErr error
		gotErr := bucket.Add(contact)
		if gotErr != wantErr {
			t.Errorf("wanted err=%v got err =%v", wantErr, gotErr)
		}
		gotErr = bucket.Add(contact)
		if gotErr != wantErr {
			t.Errorf("wanted err=%v got err =%v", wantErr, gotErr)
		}
		contacts := bucket.root.contacts
		if bytes.Equal(contacts[0].GetID(), contactID) == false {
			t.Errorf("wanted contact=%v got =%v", contactID, contacts[0].GetID())
		}
		if len(contacts) != 1 {
			t.Errorf("wanted contact.len=%v got =%v", 1, len(contacts))
		}
	})

	t.Run("adding same contact moves it to the end of the root node (most-recently-contacted end)", func(t *testing.T) {
		bucket := New(localID)

		bucket.Add(&KContact{id: makeIDStr("a")})
		contacts := bucket.root.contacts
		if len(contacts) != 1 {
			t.Errorf("wanted contact.len=%v got =%v", 1, len(contacts))
		}
		bucket.Add(&KContact{id: makeIDStr("b")})
		contacts = bucket.root.contacts
		if len(contacts) != 2 {
			t.Errorf("wanted contact.len=%v got =%v", 2, len(contacts))
		}
		if bytes.Equal(contacts[0].GetID(), makeIDStr("a")) == false { // least-recently-contacted end
			t.Errorf("wanted contacts[0]=%v got=%v", makeIDStr("a"), contacts[0].GetID())
		}
		bucket.Add(&KContact{id: makeIDStr("a")})
		contacts = bucket.root.contacts
		if len(contacts) != 2 {
			t.Errorf("wanted contact.len=%v got =%v", 2, len(contacts))
		}
		if bytes.Equal(contacts[0].GetID(), makeIDStr("b")) == false {
			t.Errorf("wanted contacts[0]=%v got=%v", makeIDStr("b"), contacts[0].GetID())
		}
		if bytes.Equal(contacts[1].GetID(), makeIDStr("a")) == false { // most-recently-contacted end
			t.Errorf("wanted contacts[1]=%v got=%v", makeIDStr("a"), contacts[0].GetID())
		}
	})

	t.Run("adding contact to bucket that can't be split results in calling 'ping' callback", func(t *testing.T) {
		bucket := New(localID)
		var called bool
		bucket.OnPing(func(contacts []ContactIdentifier, replacement ContactIdentifier) {
			if len(contacts) != bucket.numberOfNodesToPing {
				t.Errorf("wanted len(contacts)=%v got=%v", bucket.numberOfNodesToPing, len(contacts))
			}
			rightContacts := bucket.root.right.contacts
			var i int
			for i = 0; i < bucket.numberOfNodesToPing; i++ {
				// the least recently contacted end of the node should be pinged
				if bytes.Equal(contacts[i].GetID(), rightContacts[i].GetID()) == false {
					t.Errorf("wanted contact.id=%v got=%v", contacts[i].GetID(), rightContacts[i].GetID())
				}
			}
			b := makeID(0x80, byte(bucket.numberOfNodesPerKBucket))
			if bytes.Equal(replacement.GetID(), b) == false {
				t.Errorf("wanted replacement.id=%v got=%v", b, replacement.GetID())
			}
			called = true
		})
		for i := 0; i < bucket.numberOfNodesPerKBucket+1; i++ {
			bucket.Add(&KContact{id: makeID(0x80, byte(i))})
		}
		<-time.After(time.Millisecond)
		if !called {
			panic("not called.")
		}
	})

	t.Run("should generate event 'added' once", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: makeIDStr("a")}
		var i int64
		bucket.OnAdded(func(added ContactIdentifier) {
			atomic.AddInt64(&i, 1)
			if bytes.Equal(added.GetID(), contact.GetID()) == false {
				t.Errorf("wanted contact.id=%v got=%v", contact.GetID(), added.GetID())
			}
		})
		bucket.Add(contact)
		bucket.Add(contact)
		<-time.After(time.Millisecond)
		if i != 1 {
			t.Errorf("wanted onadded calls=%v got=%v", 1, i)
		}
	})

	t.Run("should generate event 'added' when adding to a split node", func(t *testing.T) {
		bucket := New(makeID())
		for i := 0; i < bucket.numberOfNodesPerKBucket+1; i++ {
			bucket.Add(&KContact{id: makeIDStr(fmt.Sprintf("%v", i))})
		}
		if bucket.root.contacts != nil {
			t.Errorf("Wanted root.contacts=%v, got=%v", nil, bucket.root.contacts)
		}
		contact := &KContact{id: makeIDStr("a")}
		var called bool
		bucket.OnAdded(func(added ContactIdentifier) {
			called = true
			if bytes.Equal(added.GetID(), contact.GetID()) == false {
				t.Errorf("wanted contact.id=%v got=%v", contact.GetID(), added.GetID())
			}
		})
		bucket.Add(contact)
		<-time.After(time.Millisecond)
		if called == false {
			panic("not called.")
		}
	})
}

func TestCount(t *testing.T) {
	t.Run("count returns 0 when no contacts in bucket", func(t *testing.T) {
		bucket := New(localID)
		want := 0
		got := bucket.Count()
		if got != want {
			t.Errorf("wanted Count()=%v got=%v", want, got)
		}
	})
	t.Run("count returns 1 when 1 contact in bucket", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: makeIDStr("a")}
		bucket.Add(contact)
		want := 1
		got := bucket.Count()
		if got != want {
			t.Errorf("wanted Count()=%v got=%v", want, got)
		}
	})
	t.Run("count returns 1 when same contact added to bucket twice", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: makeIDStr("a")}
		bucket.Add(contact)
		bucket.Add(contact)
		want := 1
		got := bucket.Count()
		if got != want {
			t.Errorf("wanted Count()=%v got=%v", want, got)
		}
	})
	t.Run("count returns number of added unique contacts", func(t *testing.T) {
		bucket := New(localID)
		bucket.Add(&KContact{id: makeIDStr("a")})
		bucket.Add(&KContact{id: makeIDStr("a")})
		bucket.Add(&KContact{id: makeIDStr("b")})
		bucket.Add(&KContact{id: makeIDStr("b")})
		bucket.Add(&KContact{id: makeIDStr("c")})
		bucket.Add(&KContact{id: makeIDStr("d")})
		bucket.Add(&KContact{id: makeIDStr("c")})
		bucket.Add(&KContact{id: makeIDStr("d")})
		bucket.Add(&KContact{id: makeIDStr("e")})
		bucket.Add(&KContact{id: makeIDStr("f")})
		want := 6
		got := bucket.Count()
		if got != want {
			t.Errorf("wanted Count()=%v got=%v", want, got)
		}
	})
}

func TestIndexOf(t *testing.T) {
	t.Run("indexOf returns a contact with id that contains the same byte sequence as the test contact", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: makeIDStr("a")}
		bucket.Add(contact)
		want := 0
		got := bucket.IndexOf(bucket.root, contact.id)
		if got != want {
			t.Errorf("wanted IndexOf()=%v got=%v", want, got)
		}
	})
	t.Run("indexOf returns -1 if contact is not found", func(t *testing.T) {
		bucket := New(localID)
		bucket.Add(&KContact{id: makeIDStr("a")})
		want := -1
		got := bucket.IndexOf(bucket.root, makeIDStr("b"))
		if got != want {
			t.Errorf("wanted IndexOf()=%v got=%v", want, got)
		}
	})
}

func TestSplit(t *testing.T) {
	t.Run("adding a contact does not split node", func(t *testing.T) {
		bucket := New(localID)
		bucket.Add(&KContact{id: makeIDStr("a")})
		if bucket.root.left != nil {
			t.Errorf("Wanted bucket.root.left=%v got=%v", nil, bucket.root.right)
		}
		if bucket.root.right != nil {
			t.Errorf("Wanted bucket.root.right=%v got=%v", nil, bucket.root.right)
		}
		if bucket.root.contacts == nil {
			t.Errorf("Wanted bucket.root.contacts=not %v got=%v", nil, bucket.root.contacts)
		}
	})
	t.Run("adding maximum number of contacts (per node) [20] into node does not split node", func(t *testing.T) {
		bucket := New(localID)
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			bucket.Add(&KContact{id: makeID(byte(i))})
		}
		if bucket.root.left != nil {
			t.Errorf("Wanted bucket.root.left=%v got=%v", nil, bucket.root.right)
		}
		if bucket.root.right != nil {
			t.Errorf("Wanted bucket.root.right=%v got=%v", nil, bucket.root.right)
		}
		if bucket.root.contacts == nil {
			t.Errorf("Wanted bucket.root.contacts=not %v got=%v", nil, bucket.root.contacts)
		}
	})
	t.Run("adding maximum number of contacts (per node) + 1 [21] into node splits the node", func(t *testing.T) {
		bucket := New(localID)
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket+1; i++ {
			bucket.Add(&KContact{id: makeID(byte(i))})
		}
		if bucket.root.contacts != nil {
			t.Errorf("Wanted bucket.root.contacts=%v got=%v", nil, bucket.root.contacts)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("removing a contact should remove contact from nested buckets", func(t *testing.T) {
		bucket := New(makeID(0x00, 0x00))
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			bucket.Add(&KContact{id: makeID(0x80, byte(i))}) // make sure all go into "far away" bucket
		}
		// cause a split to happen
		bucket.Add(&KContact{id: makeID(0x00, byte(i))})

		toDelete := &KContact{id: makeID(0x80, 0x00)}
		want := 0
		got := bucket.IndexOf(bucket.root.right, toDelete.id)
		if got != want {
			t.Errorf("wanted IndexOf()=%v got=%v", want, got)
		}
		deleted := bucket.Remove(toDelete.id)
		if deleted == nil {
			t.Errorf("wanted Remove()=%v got=%v", toDelete, deleted)
		} else if bytes.Equal(deleted.GetID(), toDelete.GetID()) == false {
			t.Errorf("wanted Remove().id=%v got.id=%v", toDelete.GetID(), deleted.GetID())
		}
		want = -1
		got = bucket.IndexOf(bucket.root.right, toDelete.GetID())
		if got != want {
			t.Errorf("wanted IndexOf()=%v got=%v", want, got)
		}
	})
	t.Run("should generate 'removed'", func(t *testing.T) {
		bucket := New(localID)
		contact := &KContact{id: makeIDStr("a")}
		bucket.Add(contact)
		var called bool
		bucket.OnRemoved(func(removed ContactIdentifier) {
			if bytes.Equal(removed.GetID(), contact.GetID()) == false {
				t.Errorf("wanted OnRemoved()=%v got=%v", contact, removed)
			}
			called = true
		})
		deleted := bucket.Remove(contact.id)
		if deleted == nil {
			t.Errorf("wanted Remove()=%v got=%v", contact, deleted)
		} else if bytes.Equal(deleted.GetID(), contact.GetID()) == false {
			t.Errorf("wanted Remove()=%v got=%v", contact.GetID(), deleted.GetID())
		}
		<-time.After(time.Millisecond)
		if !called {
			panic("not called")
		}
	})
	t.Run("should generate event 'removed' when removing from a split bucket", func(t *testing.T) {
		bucket := New(makeID())

		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			bucket.Add(&KContact{id: makeIDStr(fmt.Sprintf("%v", i))})
		}

		contact := &KContact{id: makeIDStr("a")}
		var called bool
		bucket.OnRemoved(func(removed ContactIdentifier) {
			if bytes.Equal(removed.GetID(), contact.GetID()) == false {
				t.Errorf("wanted OnRemoved()=%v got=%v", contact, removed)
			}
			called = true
		})
		bucket.Add(contact)
		bucket.Remove(contact.id)
	})
}

func TestToArray(t *testing.T) {
	t.Run("should return empty array if no contacts", func(t *testing.T) {
		bucket := New(makeID())
		want := 0
		got := len(bucket.ToArray())
		if got != want {
			t.Errorf("wanted ToArray()=%v got=%v", want, got)
		}
	})
	t.Run("should return all contacts in an array arranged from low to high buckets", func(t *testing.T) {
		bucket := New(makeID(0x00, 0x00))
		var expectedIds [][]byte
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			bucket.Add(&KContact{id: makeID(0x80, byte(i))})
			expectedIds = append(expectedIds, makeID(0x80, byte(i)))
		}
		bucket.Add(&KContact{id: makeID(0x00, 0x80, byte(i-1))})
		contacts := bucket.ToArray()
		{
			want := bucket.numberOfNodesPerKBucket + 1
			got := len(contacts)
			if got != want {
				t.Errorf("wanted ToArray()=%v got=%v", want, got)
			}
		}
		{
			want := makeID(0x00, 0x80, byte(i-1))
			got := contacts[0]
			if bytes.Equal(contacts[0].GetID(), want) == false {
				t.Errorf("Wanted contact[0].id=%v, got=%v", want, got.GetID())
			}
		}
		contacts = contacts[1:]
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			if bytes.Equal(contacts[i].GetID(), expectedIds[i]) == false {
				t.Errorf("Wanted contact[%v].id=%v, got=%v", i, expectedIds[i], contacts[i].GetID())
			}
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("deprecated vectorClock results in contact drop", func(t *testing.T) {
		bucket := New([]byte{})
		err := bucket.Add(&KContact{id: makeIDStr("a"), vectorClock: 3})
		if err != nil {
			t.Error(err)
		}
		err = bucket.update(bucket.root, 0, &KContact{id: makeIDStr("a"), vectorClock: 2})
		if err != nil {
			t.Error(err)
		}
		want := 3
		got := bucket.root.contacts[0].(ContactVectorProvider).GetVectorClock()
		if got != want {
			t.Errorf("wanted vectorClock()=%v got=%v", want, got)
		}
	})
	t.Run("equal vectorClock results in contact marked as most recent", func(t *testing.T) {
		bucket := New([]byte{})
		contact := &KContact{id: makeIDStr("a"), vectorClock: 3}
		err := bucket.Add(contact)
		if err != nil {
			t.Error(err)
		}
		err = bucket.Add(&KContact{id: makeIDStr("b")})
		if err != nil {
			t.Error(err)
		}
		err = bucket.update(bucket.root, 0, contact)
		if err != nil {
			t.Error(err)
		}
		resContact := bucket.root.contacts[1]
		if bytes.Equal(resContact.GetID(), contact.GetID()) == false {
			t.Errorf("wanted bucket.root.contacts[1].id=%v got=%v", contact.GetID(), resContact.GetID())
		}
		vRes := resContact.(ContactVectorProvider)
		if vRes.GetVectorClock() != contact.GetVectorClock() {
			t.Errorf("wanted bucket.root.contacts[1].vectorClock=%v got=%v", contact.vectorClock, vRes.GetVectorClock())
		}
	})
	t.Run("more recent vectorClock results in contact update and contact being marked as most recent", func(t *testing.T) {
		bucket := New([]byte{})
		contact := &KContact{id: makeIDStr("a"), vectorClock: 3, addr: &net.UDPAddr{Port: 10}}
		err := bucket.Add(contact)
		if err != nil {
			t.Error(err)
		}
		err = bucket.Add(&KContact{id: makeIDStr("b")})
		if err != nil {
			t.Error(err)
		}
		err = bucket.update(bucket.root, 0, &KContact{id: makeIDStr("a"), vectorClock: 4, addr: &net.UDPAddr{Port: 20}})
		if err != nil {
			t.Error(err)
		}
		resContact := bucket.root.contacts[1]
		if bytes.Equal(resContact.GetID(), contact.GetID()) == false {
			t.Errorf("wanted bucket.root.contacts[1].id=%v got=%v", contact.GetID(), resContact.GetID())
		}
		if resContact.(ContactVectorProvider).GetVectorClock() != 4 {
			t.Errorf("wanted bucket.root.contacts[1].vectorClock=%v got=%v", 4, contact.vectorClock)
		}
		if resContact.GetAddr().Port != 20 {
			t.Errorf("wanted bucket.root.contacts[1].addr.Port=%v got=%v", 20, resContact.GetAddr().Port)
		}
	})
	t.Run("should generate 'updated'", func(t *testing.T) {
		bucket := New(makeID())
		contact1 := &KContact{id: makeIDStr("a"), vectorClock: 1}
		contact2 := &KContact{id: makeIDStr("a"), vectorClock: 2}
		var called bool
		bucket.OnUpdated(func(oldContact, newContact ContactIdentifier) {
			if bytes.Equal(oldContact.GetID(), contact1.GetID()) == false {
				t.Errorf("wanted oldContact.id=%v got=%v", oldContact.GetID(), contact1.GetID())
			}
			if bytes.Equal(newContact.GetID(), contact2.GetID()) == false {
				t.Errorf("wanted newContact.id=%v got=%v", newContact.GetID(), contact2.GetID())
			}
			called = true
		})
		err := bucket.Add(contact1)
		if err != nil {
			t.Error(err)
		}
		err = bucket.Add(contact2)
		if err != nil {
			t.Error(err)
		}
		<-time.After(time.Millisecond)
		if called == false {
			panic("not called.")
		}
	})
	t.Run("should generate event 'updated' when updating a split node", func(t *testing.T) {
		bucket := New(makeID())
		for i := 0; i < bucket.numberOfNodesPerKBucket+1; i++ {
			bucket.Add(&KContact{id: makeIDStr(fmt.Sprintf("%v", i))})
		}
		contact1 := &KContact{id: makeIDStr("a"), vectorClock: 1}
		contact2 := &KContact{id: makeIDStr("a"), vectorClock: 2}
		var called bool
		bucket.OnUpdated(func(oldContact, newContact ContactIdentifier) {
			if bytes.Equal(oldContact.GetID(), contact1.GetID()) == false {
				t.Errorf("wanted oldContact.id=%v got=%v", oldContact.GetID(), contact1.GetID())
			}
			if bytes.Equal(newContact.GetID(), contact2.GetID()) == false {
				t.Errorf("wanted newContact.id=%v got=%v", newContact.GetID(), contact2.GetID())
			}
			called = true
		})
		err := bucket.Add(contact1)
		if err != nil {
			t.Error(err)
		}
		err = bucket.Add(contact2)
		if err != nil {
			t.Error(err)
		}
		<-time.After(time.Millisecond)
		if called == false {
			panic("not called.")
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("get retrieves null if no contacts", func(t *testing.T) {
		bucket := New(makeID())
		var wanted ContactIdentifier
		got := bucket.Get(makeIDStr("a"))
		if got != wanted {
			t.Errorf("Wanted contact=%v got=%v", wanted, got)
		}
	})
	t.Run("get retrieves a contact that was added", func(t *testing.T) {
		bucket := New(makeID())
		wanted := &KContact{id: makeIDStr("a")}
		err := bucket.Add(wanted)
		if err != nil {
			panic(err)
		}
		got := bucket.Get(makeIDStr("a"))
		if got == nil {
			t.Errorf("Wanted contact=%v got=%v", wanted.id, got)
		} else if bytes.Equal(got.GetID(), wanted.GetID()) == false {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got.GetID())
		}
	})
	t.Run("get retrieves most recently added contact if same id", func(t *testing.T) {
		bucket := New(makeID())
		contact := &KContact{id: makeIDStr("a"), vectorClock: 0, addr: &net.UDPAddr{Port: 1}}
		contact2 := &KContact{id: makeIDStr("a"), vectorClock: 1, addr: &net.UDPAddr{Port: 2}}
		wanted := contact2
		err := bucket.AddMany(contact, contact2)
		if err != nil {
			panic(err)
		}
		got := bucket.Get(makeIDStr("a"))
		if got == nil {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got.GetID())
		} else if bytes.Equal(got.GetID(), wanted.GetID()) == false {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got.GetID())
		} else if wanted.GetAddr().Port != got.GetAddr().Port {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got.GetID())
		}
	})
	t.Run("get retrieves contact from nested leaf node", func(t *testing.T) {
		bucket := New(makeID(0x00, 0x00))
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			bucket.Add(&KContact{id: makeID(0x80, byte(i)), addr: &net.UDPAddr{Port: i}})
		}
		wanted := &KContact{id: makeID(0x00, byte(i)), addr: &net.UDPAddr{Port: i}}
		bucket.Add(wanted) // cause a split to happen
		got := bucket.Get(wanted.id)
		if got == nil {
			t.Errorf("Wanted contact=%v got=%v", wanted.id, got)
		} else if bytes.Equal(got.GetID(), wanted.GetID()) == false {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got)
		} else if wanted.GetAddr().Port != got.GetAddr().Port {
			t.Errorf("Wanted contact=%v got=%v", wanted.GetID(), got)
		}
	})
}

func TestClosest(t *testing.T) {
	t.Run("closest nodes are returned", func(t *testing.T) {
		bucket := New(makeID())
		var i int
		for i = 0; i < 0x12; i++ {
			err := bucket.Add(&KContact{id: makeID(byte(i)), addr: &net.UDPAddr{Port: i}})
			if err != nil {
				panic(err)
			}
		}
		contacts := bucket.Closest(makeID(0x15), 3)
		if len(contacts) != 3 {
			t.Errorf("Wanted len(contacts)=%v got=%v", 3, len(contacts))
			return
		}
		if bytes.Equal(contacts[0].GetID(), makeID(0x11)) == false { // distance: 00000100
			t.Errorf("Wanted contacts[0].id=%v got=%v", makeID(0x11), contacts[0].GetID())
		}
		if bytes.Equal(contacts[1].GetID(), makeID(0x10)) == false { // distance: 00000101
			t.Errorf("Wanted contacts[1].id=%v got=%v", makeID(0x10), contacts[1].GetID())
		}
		if bytes.Equal(contacts[2].GetID(), makeID(0x05)) == false { // distance: 00010000
			t.Errorf("Wanted contacts[2].id=%v got=%v", makeID(0x05), contacts[2].GetID())
		}
	})
	t.Run("closest nodes are returned (including exact match)", func(t *testing.T) {
		bucket := New(makeID())
		var i int
		for i = 0; i < 0x12; i++ {
			err := bucket.Add(&KContact{id: makeID(byte(i)), addr: &net.UDPAddr{Port: i}})
			if err != nil {
				panic(err)
			}
		}
		wanted := &KContact{id: makeID(0x11), addr: &net.UDPAddr{Port: i}} // 00010001
		contacts := bucket.Closest(wanted.id, 3)
		if len(contacts) != 3 {
			t.Errorf("Wanted len(contacts)=%v got=%v", 3, len(contacts))
			return
		}
		if bytes.Equal(contacts[0].GetID(), makeID(0x11)) == false { // distance: 00000000
			t.Errorf("Wanted contacts[0].id=%v got=%v", makeID(0x11), contacts[0].GetID())
		}
		if bytes.Equal(contacts[1].GetID(), makeID(0x10)) == false { // distance: 00000001
			t.Errorf("Wanted contacts[1].id=%v got=%v", makeID(0x10), contacts[1].GetID())
		}
		if bytes.Equal(contacts[2].GetID(), makeID(0x01)) == false { // distance: 00010000
			t.Errorf("Wanted contacts[2].id=%v got=%v", makeID(0x01), contacts[2].GetID())
		}
	})
	t.Run("closest nodes are returned even if there isn't enough in one bucket", func(t *testing.T) {
		bucket := New(makeID(0x00, 0x00))
		var i int
		for i = 0; i < bucket.numberOfNodesPerKBucket; i++ {
			err := bucket.Add(&KContact{id: makeID(0x80, byte(i)), addr: &net.UDPAddr{Port: i}})
			if err != nil {
				panic(err)
			}
			err = bucket.Add(&KContact{id: makeID(0x01, byte(i)), addr: &net.UDPAddr{Port: i * 2}})
			if err != nil {
				panic(err)
			}
		}
		err := bucket.Add(&KContact{id: makeID(0x00, 0x01), addr: &net.UDPAddr{Port: i*2 + 1}})
		if err != nil {
			panic(err)
		}
		wanted := &KContact{id: makeID(0x00, 0x03), addr: &net.UDPAddr{Port: i*2 + 2}} // 0000000000000011
		contacts := bucket.Closest(wanted.id, 22)
		if len(contacts) != 22 {
			t.Errorf("Wanted len(contacts)=%v got=%v", 22, len(contacts))
		}
		if bytes.Equal(contacts[0].GetID(), makeID(0x00, 0x01)) == false { // distance: 0000000000000010
			t.Errorf("Wanted contacts[0].id=%v got=%v", makeID(0x00, 0x01), contacts[0].GetID())
		}
		if bytes.Equal(contacts[1].GetID(), makeID(0x01, 0x03)) == false { // distance: 0000000100000000
			t.Errorf("Wanted contacts[1].id=%v got=%v", makeID(0x01, 0x03), contacts[1].GetID())
		}
		if bytes.Equal(contacts[2].GetID(), makeID(0x01, 0x02)) == false { // distance: 0000000100000010
			t.Errorf("Wanted contacts[2].id=%v got=%v", makeID(0x01, 0x02), contacts[2].GetID())
		}
		if bytes.Equal(contacts[3].GetID(), makeID(0x01, 0x01)) == false {
			t.Errorf("Wanted contacts[3].id=%v got=%v", makeID(0x01, 0x01), contacts[3].GetID())
		}
		if bytes.Equal(contacts[4].GetID(), makeID(0x01, 0x00)) == false {
			t.Errorf("Wanted contacts[4].id=%v got=%v", makeID(0x01, 0x00), contacts[4].GetID())
		}
		if bytes.Equal(contacts[5].GetID(), makeID(0x01, 0x07)) == false {
			t.Errorf("Wanted contacts[5].id=%v got=%v", makeID(0x01, 0x07), contacts[5].GetID())
		}
		if bytes.Equal(contacts[6].GetID(), makeID(0x01, 0x06)) == false {
			t.Errorf("Wanted contacts[6].id=%v got=%v", makeID(0x01, 0x06), contacts[6].GetID())
		}
		if bytes.Equal(contacts[7].GetID(), makeID(0x01, 0x05)) == false {
			t.Errorf("Wanted contacts[7].id=%v got=%v", makeID(0x01, 0x05), contacts[7].GetID())
		}
		if bytes.Equal(contacts[8].GetID(), makeID(0x01, 0x04)) == false {
			t.Errorf("Wanted contacts[8].id=%v got=%v", makeID(0x01, 0x04), contacts[8].GetID())
		}
		if bytes.Equal(contacts[9].GetID(), makeID(0x01, 0x0b)) == false {
			t.Errorf("Wanted contacts[9].id=%v got=%v", makeID(0x01, 0x0b), contacts[9].GetID())
		}
		if bytes.Equal(contacts[10].GetID(), makeID(0x01, 0x0a)) == false {
			t.Errorf("Wanted contacts[10].id=%v got=%v", makeID(0x01, 0x0a), contacts[10].GetID())
		}
		if bytes.Equal(contacts[11].GetID(), makeID(0x01, 0x09)) == false {
			t.Errorf("Wanted contacts[11].id=%v got=%v", makeID(0x01, 0x09), contacts[11].GetID())
		}
		if bytes.Equal(contacts[12].GetID(), makeID(0x01, 0x08)) == false {
			t.Errorf("Wanted contacts[12].id=%v got=%v", makeID(0x01, 0x08), contacts[12].GetID())
		}
		if bytes.Equal(contacts[13].GetID(), makeID(0x01, 0x0f)) == false {
			t.Errorf("Wanted contacts[133].id=%v got=%v", makeID(0x01, 0x0f), contacts[13].GetID())
		}
		if bytes.Equal(contacts[14].GetID(), makeID(0x01, 0x0e)) == false {
			t.Errorf("Wanted contacts[14].id=%v got=%v", makeID(0x01, 0x0e), contacts[14].GetID())
		}
		if bytes.Equal(contacts[15].GetID(), makeID(0x01, 0x0d)) == false {
			t.Errorf("Wanted contacts[15].id=%v got=%v", makeID(0x01, 0x0d), contacts[15].GetID())
		}
		if bytes.Equal(contacts[16].GetID(), makeID(0x01, 0x0c)) == false {
			t.Errorf("Wanted contacts[16].id=%v got=%v", makeID(0x01, 0x0c), contacts[16].GetID())
		}
		if bytes.Equal(contacts[17].GetID(), makeID(0x01, 0x13)) == false {
			t.Errorf("Wanted contacts[17].id=%v got=%v", makeID(0x01, 0x13), contacts[17].GetID())
		}
		if bytes.Equal(contacts[18].GetID(), makeID(0x01, 0x12)) == false {
			t.Errorf("Wanted contacts[18].id=%v got=%v", makeID(0x01, 0x12), contacts[18].GetID())
		}
		if bytes.Equal(contacts[19].GetID(), makeID(0x01, 0x11)) == false {
			t.Errorf("Wanted contacts[19].id=%v got=%v", makeID(0x01, 0x11), contacts[19].GetID())
		}
		if bytes.Equal(contacts[20].GetID(), makeID(0x01, 0x10)) == false {
			t.Errorf("Wanted contacts[22].id=%v got=%v", makeID(0x01, 0x10), contacts[20].GetID())
		}
		if bytes.Equal(contacts[21].GetID(), makeID(0x80, 0x03)) == false { // distance: 1000000000000000
			t.Errorf("Wanted contacts[21].id=%v got=%v", makeID(0x80, 0x03), contacts[21].GetID())
		}
	})
}
