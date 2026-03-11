# shelldoc: Test Unix shell commands in Markdown documentation

[![CI](https://github.com/mirkoboehm/shelldoc/actions/workflows/ci.yml/badge.svg)](https://github.com/mirkoboehm/shelldoc/actions/workflows/ci.yml)

Documentation lies. Not intentionally, however code evolves and docs get
stale. ``shelldoc`` keeps your Markdown honest by executing the shell
commands in your documentation and verifying they still work.

## Why shelldoc?

<img src="assets/shelldoc-logo.png" alt="shelldoc terminal output" width="10%" align="right">


- **Language-agnostic**: While ``shelldoc`` is written in Go, it works
  with any Markdown documentation. It tests shell commands, not
  language-specific code, making it useful for projects in any
  language.
- **Zero dependencies**: Install with `go install` and get a single,
  self-contained binary. No runtime dependencies required.
- **CI-friendly**: Install the ``shelldoc`` binary at container build
  time. No Go toolchain needed when running your tests. The
  binary is statically linked and comes with no additional dependencies.
- **License-safe**: ``shelldoc`` only tests your documentation; it
  doesn't become part of your project. Its license never proliferates
  to your code.

## Basic usage

``shelldoc`` parses a Markdown input file, detects the code blocks in
it, executes them and compares their output with the content of the
code block. For example, the following code block contains a command,
indicated by either leading a _$_ or a _>_ trigger character, and an
expected response:

    $ echo Hello
    Hello

Lines in code blocks that begin with a ``$`` or a ``>`` _trigger character_
are considered commands. Lines in between are the expected response.
``shelldoc`` executes these commands and checks whether they succeed
and produce the expected output:

~~~shell
% shelldoc run README.md
SHELLDOC: doc-testing "README.md" ...
 CMD (1): echo Hello                                ?  Hello                      :  PASS (match)
 CMD (2): go install github.com/mirkoboehm/shel...  ?  ...                        :  PASS (match)
 CMD (3): export GREETING="Hello World"             ?  (no response expected)     :  PASS (execution successful)
 CMD (4): echo $GREETING                            ?  Hello World                :  PASS (match)
 CMD (5): echo Hello && false                       ?  Hello                      :  PASS (match)
 CMD (6): (exit 2)                                  ?  (no response expected)     :  PASS (execution successful)
 CMD (7): (sleep 1; exit 3)                         ?  (no response expected)     :  PASS (execution successful)
SUCCESS: 7 tests - 7 successful, 0 failures, 0 errors
~~~

Note that this example is not executed as a test by ``shelldoc``, since
it does not start with a trigger character. Doing so would
cause an infinite recursion when evaluating the README.md using
``shelldoc``. Try it :-) The percent symbol is commonly used as a shell
prompt next to  _$_ or a _>_. It can be used in documentation as a
prompt indicator without triggering a ``shelldoc`` test.

The XML output allows test results to be integrated into CI workflows.
Review the "Selftest Results" section for any of the workflow runs of
the pull request action in this repository for an example:

![A sample visualization of the shelldoc selftest results for this
README page](assets/shelldoc-testresults-visualization.png "selftest
results")


## Installation

The usual way to install ``shelldoc`` is using `go install`:

```shell {shelldocwhatever}
$ go install github.com/mirkoboehm/shelldoc/cmd/shelldoc@latest
...
```

Executing documentation may have side effects. For example, running
this `go install` command just installed the latest version of ``shelldoc``
in your system. Containers or VMs can be used to isolate such side
effects.

## Details and syntax

All code blocks in the Markdown input are evaluated and executed as
tests. A test succeeds if it returns the expected exit code, and the
output of the command matches the response specified in the code
block.

``shelldoc`` supports both simple and fenced code blocks. An ellipsis,
as used in the description on how to install ``shelldoc`` above,
indicates that all output is accepted from this point forward as long
as the command exits with the expected return code (zero, by default).

The `-v (--verbose)` flag enables additional diagnostic output.

A shell is launched that will execute all shell commands in a single
Markdown file. By default, the user's configured shell is used. A
different shell can be specified using the `-s (--shell)` flag:

    % shelldoc --verbose run --shell=/bin/sh README.md
	Note: Using user-specified shell /bin/sh.
	...

The shell's lifetime is that of the test run of a single Markdown
file. The environment of the shell is available between test
interactions:

	$ export GREETING="Hello World"
	$ echo $GREETING
	Hello World

``shelldoc`` uses the [Blackfriday Markdown processor](https://github.com/russross/blackfriday)
to parse Markdown files, and [Cobra](https://github.com/spf13/cobra) for
command line argument parsing.

## Options

Regular code blocks do not have a way to specify options. The only
thing that can be specified about them are the commands and the
responses. That means the expected return code must always be zero for
the test to succeed.

Sometimes, however, things are more complicated. Some commands are
expected to return a different exit code than zero. Some commands
return exit codes that are unknown up-front. Both options can be
handled by specifying tests in fenced code blocks. Fenced code blocks
may have an info string after the opening characters. This info string
is typically used to specify the language of the listed code. After
the language specifier however, other information may
follow. `shelldoc` uses this opportunity to allow the user to specify
options about the test. These options are:

	```shell {shelldocwhatever}
    % echo Hello && false
    Hello
    ```
Try executing this test:

```shell {shelldocwhatever}
> echo Hello && false
Hello
```

The _shelldocwhatever_ option tells ``shelldoc`` that the exit code of
the following command does not matter. If any expected response is
specified, it will still be evaluated. The test succeeds if the expected
response is produced, no matter the exit code of the command. 

An expected exit code is specified using the _shelldocexitcode_ option:

    ```shell {shelldocexitcode=2}
    % (exit 2)
    ```

This means the test is considered successful if it produces no response and returns 2.

```shell {shelldocexitcode=2}
> (exit 2)
```

The _shelldocexitcode_ specifies an exact exit code that is
expected. The test fails if the exit code of the command does not
match the specified one, or if the response does not match the
expected response.

A timeout can be specified using the `--timeout` flag or per code
block using the _shelldoctimeout_ option (in seconds). If a command
exceeds its timeout, the test fails and the test run is aborted:

    ```shell {shelldoctimeout=5 shelldocexitcode=3}
    % (sleep 1; exit 3)
    ```

This command must complete within 5 seconds and exit with code 3:

```shell {shelldoctimeout=5 shelldocexitcode=3}
> (sleep 1; exit 3)
```

## Output formats and integration into CI systems

By default, ``shelldoc`` produces human-readable output. For CI
integration, use the ``--xml`` flag to generate results in _JUnit XML_
format, which most CI systems understand natively.

## Limitations

- **Shared shell session**: All commands in a Markdown file run in the
  same shell session, allowing commands to depend on each other (e.g.,
  setting variables, changing directories). Be aware that earlier
  commands affect the environment for later ones.
- **Sequential processing**: Files are processed one at a time.
  Parallel execution is not supported.

## Security considerations

``shelldoc`` executes shell commands from Markdown files. This has
security implications:

- **Commands run with your permissions**: Any command in the Markdown
  file executes with the same privileges as the user running
  ``shelldoc``. If you are not sure you can trust the documentation,
  only run ``shelldoc`` in a contained environment, e.g., a container
  or a VM.
- **Environment persistence**: Environment variables and shell state
  persist between commands within a file. This is by design, but be
  aware that earlier commands can affect later ones.
- **No sandboxing**: Commands can modify the filesystem, network, and
  system state. Use containers or VMs when testing untrusted
  documentation.

## Contributing

``shelldoc`` is [free and open source software](https://en.wikipedia.org/wiki/Free_and_open-source_software).
Contributions are welcome. You are encouraged to install and use it, fork it for development, submit pull requests, or open issues
in the [issue tracker](https://github.com/mirkoboehm/shelldoc/issues).

To report a bug, submit a Markdown file showing what ``shelldoc`` does
versus what you expected. Bonus points if it's minimal and reproducible.

## Authors and license

``shelldoc`` was developed by [Mirko Boehm](https://creative-destruction.org).

The command line programs of ``shelldoc`` are located in the `cmd/`
subdirectory and licensed under the terms of the GPL, version 3. The
reusable components are located in the `pkg/` subdirectory and
licensed under the terms of the LGPL version 3. Unit test and example
code is licensed under the Apache-2.0 license.
