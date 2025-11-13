Terminal 1: go run . -id=1 -addr="127.0.0.1:5001" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100

Terminal 2: go run . -id=2 -addr="127.0.0.1:5002" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100

Terminal 3: go run . -id=3 -addr="127.0.0.1:5003" -peers="1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003" -leader=1 -duration=100

The auction will last 100 seconds, decided by the duration parameter, which in this case is 100.
After all nodes are listening on their respective ports, run this in terminal 4:
- go run . -client -op bid -caddr 127.0.0.1:5001 -bidder nick -amount 50
  Expected output should be: Bid response: status=SUCCESS, message="bid accepted"

We can also try to bid higher than the previous bid using this command:
- go run . -client -op bid -caddr 127.0.0.1:5002 -bidder nick -amount 80
  Expected output: Bid response: status=SUCCESS, message="bid accepted"

Secondly, we can try to bid lower than the previous bid which would result in a fail, this is on purpose.
- go run . -client -op bid -caddr 127.0.0.1:5003 -bidder nick -amount 40
  Expected output: Bid response: status=FAIL, message="bid is not higher than current highest bid"

Lastly, if we were to bid after the auction has finished using just one of the previous bid commands we should be met with a fail.
Expected output: Bid response: status=FAIL, message="auction has finished"

When these commands have been written in terminal 4, we can query the auction state using the following command:
- go run . -client -op result -caddr 127.0.0.1:5001
  Expected output when written within 100 seconds:
  Result:
  finished      = false
  highestBidder = "nick"
  highestAmount = 80
  message       = "auction still running"

Expected output when written after 100 seconds:
Result:
finished      = true
highestBidder = "nick"
highestAmount = 80
message       = "auction finished, winner is nick with bid 80"

This is assuming that you've copy/pasted the previous bids as well of course.