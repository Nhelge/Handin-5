Terminal 1: go run . -id=1 -addr="127.0.0.1:5001" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100

Terminal 2: go run . -id=2 -addr="127.0.0.1:5002" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100

Terminal 3: go run . -id=3 -addr="127.0.0.1:5003" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100
