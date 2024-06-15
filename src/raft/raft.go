package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
var FOLLOWER = 0
var LEADER = 1
var CANDIDATE = 2
var baseDelay = 300

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	// TODO: ADD state,term
	state         int // 0 follower, 1 leader, 2 candidate
	term          int
	lastHeartbeat time.Time
	voteCount     int
	LeaderId      int
	Log           []LogEntry
	votedFor      map[int]int
	logger        *log.Logger
	matchIndex    []int
	nextIndex     []int
	commitIndex   int
	lastApplied   int
	applyCh       chan ApplyMsg
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

type LogEntry struct {
	Term    int
	Command interface{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.term
	isleader = rf.state == LEADER
	rf.mu.Unlock()
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.lastHeartbeat = time.Now()
	rf.logger.Printf("(%d){%d}[RequestVote] node %d receive request-vote from %d term: %d\n", rf.me, rf.term, rf.me, args.CandidateId, args.Term)

	// 拒绝投票的情况：请求的任期不高于当前任期
	if rf.term > args.Term {
		reply.VoteGranted = false
		reply.Term = rf.term
		return
	}

	// 更新任期和状态，转变为跟随者
	if args.Term > rf.term {
		rf.term = args.Term
		rf.state = FOLLOWER // 假设 state = 0 表示跟随者
		reply.VoteGranted = true
		rf.votedFor[args.Term] = args.CandidateId
		rf.logger.Printf("(%d){%d}[RequestVote] term updated to %d, state changed to %d\n", rf.me, rf.term, rf.term, rf.state)
	}

	// 考虑授予投票的条件
	currentVote, votedThisTerm := rf.votedFor[args.Term]
	if !votedThisTerm || currentVote == args.CandidateId {
		rf.votedFor[args.Term] = args.CandidateId
		reply.VoteGranted = true
		reply.Term = rf.term
	} else {
		// 如果在当前任期已经投票给其他候选人，则拒绝
		reply.VoteGranted = false
		reply.Term = rf.term
	}

}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.logger.Printf("(%d){%d}[AppendEntries] term: %d, leader: %d, current leader %d\n",
		rf.me, rf.term, rf.term, args.LeaderId, rf.LeaderId)

	rf.lastHeartbeat = time.Now()
	reply.Success = true

	if args.Term > rf.term {
		rf.logger.Printf("(%d){%d}[AppendEntries] Updating term from %d to %d, new leader: %d\n",
			rf.me, rf.term, rf.term, args.Term, args.LeaderId)
		rf.state = 0 // 假设 state = 0 表示跟随者
		rf.term = args.Term
		rf.LeaderId = args.LeaderId
		return
	} else if args.Term < rf.term {
		reply.Term = rf.term
		reply.Success = false
		return
	} else if args.LeaderId != rf.LeaderId {
		rf.logger.Printf("(%d){%d}[AppendEntries] Correcting leader from %d to %d under same term %d\n",
			rf.me, rf.term, rf.LeaderId, args.LeaderId, rf.term)
		rf.LeaderId = args.LeaderId
		return
	}
	// TODO: handel entries
	rf.logger.Printf("(%d){%d}[AppendEntries] Received heartbeat from leader%d\n", rf.me, rf.term, args.LeaderId)
	rf.logger.Printf("(%d){%d}[AppendEntries] Leader's Append log: %v\n", rf.me, rf.term, args.Entries)

	if args.PrevLogIndex > len(rf.Log)-1 {
		rf.logger.Printf("(%d){%d}[AppendEntries] PrevLogIndex %d > len(rf.Log) %d\n", rf.me, rf.term, args.PrevLogIndex, len(rf.Log))
		reply.Success = false
		return

	}
	if args.PrevLogIndex >= 0 && rf.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
		rf.logger.Printf("(%d){%d}[AppendEntries] Log mismatch at index %d, args.PrevLogIndex: %d, args.PrevLogTerm: %d\n",
			rf.me, rf.term, args.PrevLogIndex, args.PrevLogIndex, args.PrevLogTerm)
		reply.Success = false
		return
	}
	if len(args.Entries) > 0 {
		rf.logger.Printf("(%d){%d}[AppendEntries] Received %d entries from leader %d\n",
			rf.me, rf.term, len(args.Entries), args.LeaderId)
		rf.Log = append(rf.Log, args.Entries...)
	}
	rf.logger.Printf("(%d){%d}[AppendEntries] Log after appending: %v\n", rf.me, rf.term, rf.Log)
	rf.logger.Printf("(%d){%d}[AppendEntries] CommitIndex: %d, LeaderCommit: %d\n", rf.me, rf.term, rf.commitIndex, args.LeaderCommit)
	rf.commitIndex = args.LeaderCommit

	reply.Success = true
	rf.logger.Printf("(%d){%d}[AppendEntries] Success,Starting to apply entries\n", rf.me, rf.term)
	rf.applyEntries()
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	rf.mu.Lock()
	rf.logger.Printf("(%d){%d}[sendRequestVote]node send vote: %d -----> %d\n", rf.me, rf.term, rf.me, server)
	rf.mu.Unlock()
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	rf.mu.Lock()
	if !ok {
		rf.logger.Printf("(%d){%d}[sendRequestVote] WARNING: Vote request to %d failed\n", rf.me, rf.term, server)
	}
	rf.mu.Unlock()
	return ok
}
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	rf.mu.Lock()
	rf.logger.Printf("(%d){%d}[sendHeartbeat] node heartbeat %d == %d ---> %d\n",
		rf.me, rf.term, args.LeaderId, rf.me, server)
	rf.mu.Unlock()
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	if !ok {
		rf.mu.Lock()
		rf.logger.Printf("(%d){%d}[sendHeartbeat] WARNING: Heartbeat to %d failed\n", rf.me, rf.term, server)
		rf.mu.Unlock()
	}
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	if rf.state != LEADER {
		term = rf.term
		rf.mu.Unlock()
		return index, term, false
	}
	term = rf.term
	rf.Log = append(rf.Log, LogEntry{term, command})
	index = len(rf.Log) - 1
	rf.logger.Printf("(%d){%d}[Start] node %d start agreement on command %v index %d\n", rf.me, rf.term, rf.me, command, index)
	rf.mu.Unlock()
	go rf.broadcastAppendEntries()

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	rf.mu.Lock()
	rf.logger.Printf("(%d){%d}[Kill]", rf.me, rf.term)
	rf.mu.Unlock()
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	heartbeatTimeout := 1 * time.Second

	for !rf.killed() {
		time.Sleep(time.Duration(baseDelay+rand.Intn(50)) * time.Millisecond)

		rf.mu.Lock()
		elapsed := time.Since(rf.lastHeartbeat)
		isTimeout := elapsed >= heartbeatTimeout
		rf.logger.Printf("(%d){%d}[ticker] current state: %d, leader %d, term %d, elapsed since last heartbeat: %v\n",
			rf.me, rf.term, rf.state, rf.LeaderId, rf.term, elapsed)
		rf.mu.Unlock()

		isLeader := rf.isLeader()
		if !isLeader && isTimeout {
			rf.mu.Lock()
			rf.logger.Printf("(%d){%d}[ticker] Heartbeat timeout for node %d. Starting election.\n", rf.me, rf.term, rf.me)
			rf.mu.Unlock()
			rf.startVote()
		}

		if rf.isLeader() {
			rf.broadcastAppendEntries()
		}

	}
}

func (rf *Raft) startVote() {

	rf.mu.Lock()
	voteTerm := rf.term + 1
	rf.term = voteTerm
	voteCandidate := rf.me
	rf.voteCount = 1
	rf.state = CANDIDATE
	rf.lastHeartbeat = time.Now()
	rf.logger.Printf("(%d){%d}[startVote]Vote start\n", rf.me, rf.term)
	rf.votedFor[voteTerm] = voteCandidate
	rf.mu.Unlock()

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(i int) {

			args := &RequestVoteArgs{
				Term:         voteTerm,
				CandidateId:  voteCandidate,
				LastLogIndex: 0,
				LastLogTerm:  0,
			}
			reply := &RequestVoteReply{}

			if !rf.sendRequestVote(i, args, reply) {
				return
			}
			rf.mu.Lock()
			if reply.Term > rf.term {
				rf.term = reply.Term
				rf.state = FOLLOWER
			}
			if reply.VoteGranted {
				rf.voteCount++
			}
			rf.mu.Unlock()
		}(i)
	}

	time.Sleep(1 * time.Second)
	rf.mu.Lock()
	rf.logger.Printf("(%d){%d}[startVote]Vote end, term %d got %d\n", rf.me, rf.term, rf.term, rf.voteCount)
	enoughVotes := rf.voteCount > len(rf.peers)/2
	rf.mu.Unlock()
	if enoughVotes {
		rf.convertToLeader()
	}
	rf.logger.Printf("(%d)[startVote]checked:%v\n", rf.me, rf.isLeader())
	rf.broadcastAppendEntries()

}

func (rf *Raft) broadcastAppendEntries() {
	rf.mu.Lock()
	term := rf.term
	leader := rf.me
	commitIndex := rf.commitIndex
	rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d broadcast, current commitIndex %d\n", rf.me, rf.term, rf.me, commitIndex)
	rf.mu.Unlock()

	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		go func(i int) {
			rf.mu.Lock()
			log := []LogEntry{}
			for j := max(0, rf.nextIndex[i]); j < len(rf.Log); j++ {
				if rf.Log[j].Command != nil {
					log = append(log, rf.Log[j])
				}
			}
			rf.logger.Printf("(%d){%d}[broadcastAppendEntries] log for node %d: %v\n", rf.me, rf.term, i, log)
			args := &AppendEntriesArgs{
				Term:         term,
				LeaderId:     leader,
				PrevLogIndex: rf.nextIndex[i] - 1,
				Entries:      log,
				LeaderCommit: commitIndex,
			}
			if args.PrevLogIndex >= 0 {
				args.PrevLogTerm = rf.Log[args.PrevLogIndex].Term
			} else {
				args.PrevLogTerm = 0
			}
			rf.mu.Unlock()

			reply := &AppendEntriesReply{}
			ok := rf.sendAppendEntries(i, args, reply)
			rf.mu.Lock()
			rf.logger.Printf("(%d){%d}[broadcastAppendEntries] reply from node %d isOK: %v\n", rf.me, rf.term, i, ok)
			rf.mu.Unlock()
			if !ok {
				rf.mu.Lock()
				rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d failed\n", rf.me, rf.term, i)
				rf.mu.Unlock()
				return
			}
			rf.mu.Lock()
			rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d ok: %v, reply.Success: %v\n", rf.me, rf.term, i, ok, reply.Success)
			rf.mu.Unlock()
			rf.handleAppendEntriesResponse(i, args, reply, ok)
			rf.mu.Lock()
			rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d done\n", rf.me, rf.term, i)
			rf.mu.Unlock()
		}(i)
	}
	time.Sleep(400 * time.Microsecond)
	rf.mu.Lock()
	rf.logger.Printf("(%d){%d}[broadcastAppendEntries] all nodes done\n", rf.me, rf.term)
	rf.mu.Unlock()

	rf.updateCommitIndex()
}

func (rf *Raft) handleAppendEntriesResponse(i int, args *AppendEntriesArgs, reply *AppendEntriesReply, ok bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if !ok {
		rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d failed, decrement nextIndex\n", rf.me, rf.term, i)
		rf.nextIndex[i] = max(1, rf.nextIndex[i]-1)
		return
	}

	if reply.Success {
		rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d success, updating matchIndex and nextIndex\n", rf.me, rf.term, i)
		rf.matchIndex[i] = args.PrevLogIndex + len(args.Entries)
		rf.nextIndex[i] = rf.matchIndex[i] + 1
	} else if reply.Term > rf.term {
		rf.logger.Printf("(%d){%d}[broadcastAppendEntries] term %d < %d become follower again.\n", rf.me, rf.term, rf.term, reply.Term)
		rf.term = reply.Term
		rf.state = 0 // Assume 0 is follower
	} else {
		rf.logger.Printf("(%d){%d}[broadcastAppendEntries] node %d append failed, decrement nextIndex\n", rf.me, rf.term, i)
		rf.nextIndex[i]--
	}
}

func (rf *Raft) updateCommitIndex() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	copyMatchIndex := make([]int, len(rf.matchIndex))
	copy(copyMatchIndex, rf.matchIndex)
	sort.Ints(copyMatchIndex)
	newCommitIndex := copyMatchIndex[(len(copyMatchIndex)-1)/2]
	rf.logger.Printf("(%d){%d}[broadcastAppendEntries] sorted matchIndex: %v\n", rf.me, rf.term, copyMatchIndex)
	rf.logger.Printf("(%d){%d}[broadcastAppendEntries] newCommitIndex: %d\n", rf.me, rf.term, newCommitIndex)
	if rf.commitIndex < newCommitIndex {
		rf.logger.Printf("(%d){%d}[broadcastAppendEntries] Updating commitIndex from %d to %d\n", rf.me, rf.term, rf.commitIndex, newCommitIndex)
		rf.commitIndex = newCommitIndex
		rf.applyEntries()
	}
	rf.logger.Printf("(%d){%d}[broadcastAppendEntries] final commitIndex: %d\n", rf.me, rf.term, rf.commitIndex)
}

func (rf *Raft) convertToLeader() {
	rf.mu.Lock()
	if rf.state == CANDIDATE {
		rf.logger.Printf("(%d){%d}[convertToLeader] LEADER \n", rf.me, rf.term)
		rf.state = 1
		rf.LeaderId = rf.me
		for i := 0; i < len(rf.peers); i++ {
			rf.nextIndex[i] = len(rf.Log)
			rf.matchIndex[i] = 0
		}

		// TODO: initialize nextIndex and matchIndex
	} else {
		rf.logger.Printf("(%d){%d}[convertToLeader] State changed to%d\n", rf.me, rf.term, rf.state)
	}

	rf.mu.Unlock()
	// rf.lastHeartbeat = time.Now()
}
func (rf *Raft) applyEntries() {
	rf.logger.Printf("(%d){%d}[applyEntries] commitIndex: %d, lastApplied: %d\n", rf.me, rf.term, rf.commitIndex, rf.lastApplied)

	if rf.commitIndex > rf.lastApplied {
		for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
			if i >= len(rf.Log) {
				break
			}
			rf.logger.Printf("(%d){%d}[applyEntries] Applying command %v at index %d\n", rf.me, rf.term, rf.Log[i].Command, i)
			if rf.Log[i].Command == nil {
				continue
			}
			msg := ApplyMsg{
				CommandValid: true,
				Command:      rf.Log[i].Command,
				CommandIndex: i,
			}
			rf.lastApplied = i
			rf.logger.Printf("(%d){%d}[applyEntries] lastApplied: %d\n", rf.me, rf.term, rf.lastApplied)
			rf.logger.Printf("(%d){%d}[applyEntries] Sending ApplyMsg: %v\n", rf.me, rf.term, msg)
			rf.applyCh <- msg
		}
	} else {
		rf.logger.Printf("(%d){%d}[applyEntries] No entries applied\n", rf.me, rf.term)
	}

}
func (rf *Raft) convertToFollower(term int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.term = term
	rf.state = 0
	rf.voteCount = 0
	// rf.lastHeartbeat = time.Now()
}

func (rf *Raft) requireHeartbeat() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.state == 0

}
func (rf *Raft) isLeader() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.state == 1

}
func (rf *Raft) heartbeatExpire(delay time.Duration) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if time.Now().Sub(rf.lastHeartbeat) > delay {
		rf.logger.Printf("(%d)[heartbeatExpire]actual delay: %.3f, expect delay: %.3f\n", rf.me, float32(time.Now().Sub(rf.lastHeartbeat))/1e9, float32(delay)/1e9)
		return true
	}
	return false
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func loggerMaker(id int) *log.Logger {
	return log.New(os.Stdout, fmt.Sprintf("INFO (worker %d): ", id), log.Ldate|log.Lmicroseconds|log.Lshortfile)
}
func (rf *Raft) Core() {
	// TODO: check state and act accordingly.

}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	go rf.Core()

	// Your initialization code here (2A, 2B, 2C).
	rf.votedFor = make(map[int]int)
	rf.logger = loggerMaker(me)
	rf.matchIndex = make([]int, len(peers))
	rf.nextIndex = make([]int, len(peers))
	rf.applyCh = applyCh
	rf.commitIndex = 0
	rf.lastApplied = 0

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.Log = append(rf.Log, LogEntry{0, nil})

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
