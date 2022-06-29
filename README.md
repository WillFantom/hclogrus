# Logrus Healthchecks.io Hook

Send logs to a [healthchecks.io](https://healthchecks.io) check.

---

## Required Configuration

 - **Check ID**: The ID for the check you wish for the logs to be associated with. This can be found in a check's ping URL.
 - **Tick Period**: Healthchecks.io expects a check to report in (contact the endpoint) at a given interval, otherwise the check is reporeted as 'down'. This plugin will send the most recent log message every interval (with a `ticker` flag set in the message JSON), so the interval must be provided.
 - **Failure Levels**: Checks can not only be considered 'down' from missing an interval, but can also be manulally flagged as 'down' (failed). The logrus log levels that, when used, should flag the check as down can be specified.

For example, if a check ID of **`example-check-1234`**, a tick period of **24 hours**, and failure levels being **Error** and **Fatal**, the following could be used:

```go
hk, _ := hclogrus.New("example-check-1234", time.Hour * 24, logrus.ErrorLevel, logrus.FatalLevel)
logrus.AddHook(hk)
```

---

## Log Entry Format

The following structure is used for JSON log entries sent to Healthchecks.io...

 - Log Level (`level`): An integer representation of the log level, where `-1` is used to spcifiy a non-logrus level used for hook system messages (such as tick, and startup).
 - Log Level String (`level_string`): An sting representation of the log level.
 - Time (`time`): The time the log entry was created (from the logrus entry).
 - Message (`message`): The message passed to logrus (from the logrus entry).
 - Data (`data`): A JSON object of the fields given to the logrus entry (from the logrus entry).
 - Ticker (`ticker`): A boolean value, that if true, means the the message is not a new entry, and instead is just a tick from the inteval ticker and will likey be repeating the last entry.
  
--- 

## Job Execution Time Monitoring

Healthchecks.io supports execution time monitoring via the `/start` suffix on the ping URL. The next message, success or failure, is then considered the end of the job (Maybe jobs should be taggable?). With this plugin, a log entry is considered a start entry if it has the field `@hc_job_start` set to `true`. This can be set like so:

```go
...
logrus.WithField(hclogrus.JobStartField, true).Infoln("Job Starting...")
...
```

The next entry will be considered the job completion entry, failure or not.

> ⚠️ Ticker messages will never be flagged as job start messages, nor will entries that will flag a failure.
