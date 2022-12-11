
- request arrives:
  - shuffle sharding (get N candidates)
  - TODO: select the queue with minimum work
  - enqueue
  - TODO: virtual clock RT increments
  - TODO: set request arrivedRT

- dispatch:
    - TODO: find the request with earliest finish time
    - dequeue
    - TODO: record RT  
