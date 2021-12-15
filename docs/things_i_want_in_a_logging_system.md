# Things I want in a Logging System

This is a vague design doc based on what I want in a logging system:

- _fast_ - should be able to saturate a 1G link assuming input and output have capacity for such production/consumption
- _configurable_ - it should allow arbitrary transformations on the incoming log data
- _safe_ - memory safe, and without the ability to kill itself with ooms etc

## Features

 - reading from sources including kafka, udp, tcp (with optional tls), files, directories/globs, and straight from JournalD
 - writing to destinations including elasticsearch, kafka, and the same system
 - reading/writing in both JSON and Binary formats (capnp)
 - Extention mechanisms with Python to allow full control over log manipulation
 - ability to handle arbitary structured messages
 - ability to send and handle backpressure if a receiver is overloaded
 - configurable buffering destinations, including disk and remote locations
 - ability for extentions to report their own metrics, as well as metrics, logs, and distributed tracing support