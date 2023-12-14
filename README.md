# dhangkanna

![kanna-attack-on-titan-min](https://github.com/khatibomar/dhangkanna/assets/35725554/d77781a3-3b8a-4947-aba1-5114d209c7f4)

dhangkanna is a distributed hang game with a Kanna theme it uses Go, Hashicorp Raft, Serf and gRPC

# prerequisites

- make
- protoc
- tsc

# Usage

to build the app run 

```sh
make build
```

after that two new binaries will be created `dhangkanna_back` and `dhangkanna_front` If on Windows they will have the extension `.exe`

to run the leader, type

```sh
./dhangkanna_back.exe -bootstrap -data-dir="C:\Users\User\AppData\Local\Temp\dhangkanna\node1" -node-name="node1"
```

you should get this output

```
distributed game: 2023/12/14 15:50:49 distributed.go:123: setting up raft
2023-12-14T15:50:49.451+0200 [INFO]  raft: initial configuration: index=0 servers=[]
2023-12-14T15:50:49.451+0200 [INFO]  raft: entering follower state: follower="Node at 127.0.0.1:4002 [Follower]" leader-address= leader-id=
distributed game: 2023/12/14 15:50:49 distributed.go:208: Done setting up raft
distributed game: 2023/12/14 15:50:49 distributed.go:40: DistributedGame initialized successfully
2023-12-14T15:50:50.944+0200 [WARN]  raft: heartbeat timeout reached, starting election: last-leader-addr= last-leader-id=
2023-12-14T15:50:50.944+0200 [INFO]  raft: entering candidate state: node="Node at 127.0.0.1:4002 [Candidate]" term=2
2023-12-14T15:50:50.946+0200 [DEBUG] raft: voting for self: term=2 id=node1
2023-12-14T15:50:50.947+0200 [DEBUG] raft: calculated votes needed: needed=1 term=2
2023-12-14T15:50:50.947+0200 [DEBUG] raft: vote granted: from=node1 term=2 tally=1
2023-12-14T15:50:50.947+0200 [INFO]  raft: election won: term=2 tally=1
2023-12-14T15:50:50.947+0200 [INFO]  raft: entering leader state: leader="Node at 127.0.0.1:4002 [Leader]"
distributed game: 2023/12/14 15:50:51 distributed.go:54: Leader found: 127.0.0.1:4002
```

bootstrap is a special flag for the first server that runs, the first server with this flag will be assigned as leader.

The default port is `4001` for serf and `4002` for gRPC and raft

to run a follower backend you need to pass the leader address so it can join.

```
./dhangkanna_back.exe -data-dir="C:\Users\User\AppData\Local\Temp\dhangkanna\node2" -node-name="node2" -bind-addr="127.0.0.1:7001" -rpc-port=7002 -start-join-addrs="127.0.0.1:4001"
```

```
distributed game: 2023/12/14 15:55:16 distributed.go:123: setting up raft
2023-12-14T15:55:16.310+0200 [INFO]  raft: initial configuration: index=4 servers="[{Suffrage:Voter ID:node1 Address:127.0.0.1:4002}]"
distributed game: 2023/12/14 15:55:16 distributed.go:208: Done setting up raft
distributed game: 2023/12/14 15:55:16 distributed.go:40: DistributedGame initialized successfully
2023-12-14T15:55:16.310+0200 [INFO]  raft: entering follower state: follower="Node at 127.0.0.1:7002 [Follower]" leader-address= leader-id=
agent: 2023/12/14 15:55:16 agent.go:155: setting up server
agent: 2023/12/14 15:55:16 agent.go:172: done setting up server
agent: 2023/12/14 15:55:16 agent.go:135: setting up discovery
agent: 2023/12/14 15:55:16 agent.go:141: [127.0.0.1:4001]
2023/12/14 15:55:16 [INFO] serf: EventMemberJoin: node2 127.0.0.1
```

to run the third node

```
./dhangkanna_back.exe -data-dir="C:\Users\User\AppData\Local\Temp\dhangkanna\node3" -node-name="node3" -bind-addr="127.0.0.1:8001" -rpc-port=8002 -start-join-addrs="127.0.0.1:4001"
```

```
distributed game: 2023/12/14 15:56:13 distributed.go:123: setting up raft
2023-12-14T15:56:13.211+0200 [INFO]  raft: initial configuration: index=0 servers=[]
2023-12-14T15:56:13.211+0200 [INFO]  raft: entering follower state: follower="Node at 127.0.0.1:8002 [Follower]" leader-address= leader-id=
distributed game: 2023/12/14 15:56:13 distributed.go:208: Done setting up raft
distributed game: 2023/12/14 15:56:13 distributed.go:40: DistributedGame initialized successfully
```

right now we have 3 servers so we achieved the quorum.

To run the frontend

```
.\dhangkanna_front.exe -backend-addr="127.0.0.1:8002"
```

```
frontend: 2023/12/14 15:58:07 main.go:72: Server is running on port 4000
```

the default port is `4000` and I picked one of the followers to initialize a connection with.

now we can visit `http://localhost:4000` to play the game. As we can see in `GIF` down below the socket is running to simulate the same state across different windows and tabs.

https://github.com/khatibomar/dhangkanna/assets/35725554/25a36b04-0b9d-4033-a46a-e034afae5231

let's launch another frontend that will point to a different follower server

```
.\dhangkanna_front.exe -port=5000 -backend-addr="127.0.0.1:7002"
```

go to `http://localhost:5000`

![image](https://github.com/khatibomar/dhangkanna/assets/35725554/f329c2e6-b0f3-45b8-8b69-f11463ad14bd)

as soon we visit the page we can see that the other followers running on port `7002` have also the same state as the leader.

https://github.com/khatibomar/dhangkanna/assets/35725554/5fbef3d1-239d-4ed1-b15e-99d7d06853fe

> I am refreshing the page manually to reflect the latest game state because as discussed in Architecture, I don't have a hook to update cross servers.

# Architecture

![image](https://github.com/khatibomar/dhangkanna/assets/35725554/03219bd0-f773-4ded-b4bc-befd586177f1)

the frontend here is served from a Go server, each frontend has a dedicated socket connection, and it will keep connecting each time you visit the address of the frontend you will get the latest state, and then newer states will be streamed for all tabs of same server.

However, to achieve synchronization between different servers, we need to have a leader who will keep all servers in sync.

When a player enters a character the gRPC load balancer will redirect the call to the leader, after that the leader will copy the state to all of the followers, then the webhook will update the frontend for other pages. All frontend updates and initialization will be handled by followers.

> The webhook is not implemented yet, so you need to manually refresh the page to get the latest state.
