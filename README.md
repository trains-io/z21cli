## The Z21 Command Line Interface

A command line utility to interact with and manage Z21 Command Station.

### Features

- Context configuration
- Query status and system information
- Monitoring and Subscription of broadcast events
- CAN bus management

### Installation

Releases are [published to GitHub](https://github.com/trains-io/z21cli/releases) where Zip, RPMs and DEBs for various operating systems can be found.

#### Installation via go install

The z21cli tool can be installed directly via `go install`. To install the latest version:

```sh
go install github.com/trains-io/z21cli@latest
```

To install a specific release:

```sh
go install github.com/trains-io/z21cli@v0.0.1
```

### Context Configuration

The `z21` CLI supports multiple named configurations. To enable a context we'll create a `demo` configuration and set it as default.

First we add a configuration to capture Z21 Command Station configuration.

```sh
z21cli context add sim --host 127.0.0.1
```

Ouput

```sh
Z21 Configuration Context "sim"

  Host: 127.0.0.1:21105

```

Next we add a context for `demo` and we use it as default.

```sh
z21cli context add demo --host 192.168.2.6 --use
```

Output

```sh
Z21 Configuration Context "demo"

  Host: 192.168.2.6:21105

```

These are the contexts, the `*` indicates the default

```sh
z21cli context ls
```

Output

```sh
Known contexts:

  ( ) sim     127.0.0.1:21105
  (*) demo    192.168.2.6:21105
```

To switch to another context:

```sh
z21cli ctx use sim
```

Use `z21cli --help` to see how to add, remove, list and show contexts.

#### Session management

The Z21 Command Station supports multiple concurrent client sessions. Each session is uniquely identified and tracked based on the client’s local IP address and UDP source port.

To show current context with its session data:

```sh
z21cli ctx show
```

Output

```sh
Z21 Configuration Context "demo"

  Host:    192.168.2.6:21105
  Session: 172.20.236.167:38991

```

To clear the session data for current context:

```sh
z21cli ctx reset
```

#### Configuration file

z21cli stores contexts in `~/.z21_contexts.json` as JSON documents.

### Query status and system information

The `z21` CLI can query the track and system status.

```sh
z21cli status
```

Output

```sh
 TRACK             STATUS (0X00) 
---------------------------------
 Emergency Stop    OFF           
 Track Voltage     ON            
 Short Circuit     OFF           
 Programming Mode  INACTIVE      

 MAIN  PROG  TEMP  SUPPLY  INTERNAL 
------------------------------------
 59mA  1mA   22°C  19.9V   18.1V
```

To print system information, like device family, hardware platform, serial number, X-bus protocol version, and  firmware version:

```sh
z21cli info
```

Output

```sh
Z21 black Z21 (2013) 265070 V4.0 1.43 [no lock]
```

Use `z21cli status -h` and `z21cli info -h` for more options.

### Monitor and Subscribe

The `z21` CLI can subscribe to Z21 broadcast events.

We will subscribe to the `SYSTEM_UPDATES` broadcast events:

```sh
z21cli sub add SYSTEM_UPDATES
```

Output

```
Subscribed to "SYSTEM_UPDATES"
```

We can now monitor broadcast events:

```
z21cli monitor
Waiting for Z21 events ...
[SYS] Main: 30mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.0V)
[SYS] Main: 36mA  Prog: 0mA   Temp: 27°C  Volt: 20.0V (18.0V)
[SYS] Main: 38mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.1V)
[SYS] Main: 33mA  Prog: 0mA   Temp: 27°C  Volt: 20.1V (18.2V)
[SYS] Main: 30mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.2V)
[SYS] Main: 54mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.2V)
[SYS] Main: 83mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.1V)
[SYS] Main: 82mA  Prog: 0mA   Temp: 27°C  Volt: 20.1V (17.9V)
```

Use `z21cli monitor -h` to see how to list and remove subscriptions.

### CAN bus management

The `z21` CLI can discover and inspect CAN bus detector devices.

To begin, we will subscribe to CAN-bus update events:

```sh
z21cli sub add CAN_DETECTOR_UPDATES
```

Output

```sh
Subscribed to "CAN_DETECTOR_UPDATES"
```

Next, we can discover or inspect devices on the CAN bus.

To run device discovery:

```sh
z21cli can discover
```

Output

```sh
Discover CAN devices (timeout: 2s) ...
 NETID   ADDR  PORT(S) 
-----------------------
 0xDB04  31    1-8
```

To inspect a specific device:

```sh
z21cli can info 0xdb04
```

Output

```sh
Device: 0xDB04 (address: 31)
 PORT  STATUS 
--------------
 1     free   
 2     free   
 3     free   
 4     free   
 5     free   
 6     free   
 7     free   
 8     free
```

### License

This project is licensed under the MIT License.

### Contributing

Contributions, bug reports, and feature requests are welcome!
Simply open an issue or submit a pull request.
