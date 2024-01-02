Run the program with `go run . sample.csv`.  If you don't have go installed but
are running an M1 mac, you should might be able to run `./logreader sample.csv`

I think it is implied by the problem statement, but just to be sure, I
interpreted the 2-minute alert period as a rolling window, rather than a check
every 2 minutes.  I keep a cyclicBuffer with hit counts for each second so that
I can subtract hit counts that are more than 2 minutes in the past and add the
recent ones as I roll the window.

To deal with the out of order log messages in the input, I chose to make the
buffer slightly larger than 2 minutes, so that I can accumulate the hit counts
for the most recent timestamps a bit before analyzing them for possible alerts.
I didn't bother to do the same for the periodic logger, because I figured it
wasn't important enough for that use case to add the complexity.

A note on variable names.  It is considered idiomatic Go to use very short
names for variables in the local scope whenever doing so is reasonably clear.
It was foreign to me when I started using Go, but I decided to follow the
practice here.  The weird spacing around + and - are also idiomatic Go.
¯\_(ツ)_/¯



Future Improvements:

* A decision I made about the periodic logger was not to bother using a heap to
  store the hit counts by section, which would have yielded a better O() run-
  time complexity when reporting the hits by section.  Go does not have
  a simple heap data structure, but I can create one by implementing
  heap.Interface in this library (https://pkg.go.dev/container/heap).  This
  could be a potential improvement, but I'd want to run performance tests
  before deciding if it helps. I suspect it would help if there are enough
  distinct sections in the logs, but might not when there are fewer.

* For my little app to be useful at a larger scale, it would have to be used
  in a distributed system - some form of map reduce architecture. The
  enlarged cyclic buffer could still be used to accumulate hits for the most
  recently reported timestamps before forwarding them to the next stage for
  aggregation.  And I would use some kind of Reporter interface similar to the
  ones I have and implement sending the hit counts to a kafka queue, or another
  communication mechanism to support downstream aggregation nodes in the
  reduction.

* If there is any problem with the timestamps in the input data, the logic
  will fall apart. If a server has a system clock that is ahead of real world
  time, it will be particularly bad.  I think it could be handled reasonably
  well by filtering outliers.  If a timestamp comes in that is too far in the
  future from the current largestTs, maybe discount it, or hold it suspect
  if subsequent logs don't agree with it.  If running in live mode, maybe the
  scrub logic could use the local system clock as a sanity check.

* If we want to eke out as much performance as possible, the regex I used to
  extract the section string could be replaced with some manual, specialized
  string parsing that would perform better.  It would be quite simpel to write.

* I don't know if the csv library I used is doing anything to scrub input
  strings for security threats, but I doubt it.  Depending if there is any
  concern that the data might be tempered with or come from unknown sources,
  then that might be something to look into as well.

