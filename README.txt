Run the program with `go run . sample.csv`.  If you don't have go installed but
are running an M1 mac, you should might be able to run `./logreader sample.csv`

I think it is implied by the problem statement, but just to be sure, I
interpreted the 2-minute alert period as a rolling window, rather than a check
every 2 minutes.  I keep a cyclicBuffer with hit counts so that I can subtract
hit counts that are more than 2 minutes in the past and add the recent ones as
I roll the window.

To deal with the out of order log messages in the input, I chose to make the
buffer slightly larger thgan 2 minutes, so that I can accumulate the hit counts
for the most recent timestamps before analyzing them for possible alerts. I
didn't bother to do the same for the periodic logger, because I figured it
wasn't important enough for that use case to add the complexity.

A note on variable names.  It is considered idiomatic Go to use very short
names for variables in the local scope whenever doing so is reasonably clear.
It was foreign to me when I started using Go, but I decided to follow the
practice here.



Future Improvements:

* For my little app to be useful at a larger scale, it would have to be used
  in a distributed system, probably a map reduce architecture. The
  PeriodicReporter and AlertReporter interfaces could be implemented to send
  the results to a kafka queue, or another communication mechanism to support
  one or more aggregation nodes in the reduction.

* Another decision I made about the periodic logger was not to bother using a
  heap to store the hit counts by section.  Go does not have a simple heap data
  structure, but I can create one by implementing heap.Interface in this
  library (https://pkg.go.dev/container/heap).  This could be a potential
  improvement, but I'd want to run performance tests before deciding if it
  helps. I suspect it would help if there are enough distinct sections in the
  logs, but might not when there are fewer.

* If there is any problem with the timestamps in the input data, the logic
  will fall apart. If a server has a system clock that is ahead of real world
  time, it will be particularly bad.  I think it could be handled reasonably
  well by filtering outliers.  If a timestamp comes in that is too far in the
  future from the current largestTs, maybe discount it, or hold it suspect
  if subsequent logs don't agree with it.  If running in live mode, maybe the
  scrub logic could use the local system clock as a sanity check.

* I don't know if the csv library I used is doing anything to scrub input
  strings for security threats, but I doubt it.  This might not be a bad idea.

