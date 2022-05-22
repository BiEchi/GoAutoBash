# GoAutoBash
This is the public repo for GoAutoBash, inherited from GoAutoGrader. This project is designed to generalize the GoAutoGrader functionalities.

## Installation

- Install Go 1.17
- Remove `report` branch in your MP repo
- Add webhook to the course repo
- `go run GoAutoGrader`

## Dir Tree

```bash
- logs - server logs
- mp1 - input space for MP1
- mp2 - input space for MP2
- mp3 - input space for MP3
- queue - producer-consumer queueing system
- server - listening server system
- templates - not used (later for user status query)
- main.go - global entry point
```
