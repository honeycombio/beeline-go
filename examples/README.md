# beeline-go examples

Each of these examples is meant to show the Honeycomb [Beeline for Go](https://docs.honeycomb.io/getting-data-in/beelines/go-beeline/) in action in that particular framework's vocabulary. The examples show simple example use of the Beeline to set up the outermost HTTP wrapper and capture some useful per-request context.

Two of these examples go a little bit further and are intended to show off more full-fledged use of the Beeline:

- `nethttp` contains an example of "everything you can do with the Beeline," including:
	- beginning a trace
	- creating extra spans within a trace
	- hooking into the Beeline's sampler function (in order to intelligently sample the events emitted)
	- using scrubber functions as callbacks to obscure sensitive metadata
	- instrumenting outbound requests (aka, what you want to use when sending things to other instrumented apps)
- `http_and_sql` contains an example of using both `nethttp` and `sql` wrappers together in a single application.
