# KLC3 Feedback Tool (Beta) Report on MP2

## Description

KLC3 feedback tool first runs the tests distributed with mp2 to you, reported in section [Easy Test](#easy-test).
If you pass all these tests, KLC3 starts symbolic execution ([what is this?](https://en.wikipedia.org/wiki/Symbolic_execution)
on your code trying to find any input (test case) to trigger your bugs. When a bug is detected, a test case will be provided to you.

We want you to resolve bugs detected before KLC3
runs time-consuming symbolic execution again, so on your next commit, KLC3 will runs all test cases previously provided
to you, in the section [Regression Test](#regression-test). If they are all passed, KLC3 will try to find new test
cases that can trigger bugs in your code, in the section [Report](#report).

KLC3 is still under test. This report can be **incorrect** or even **misleading**. If you think there is
something wrong or unclear, please contact the TAs on [Piazza](http://piazza.com/illinois/fall2020/ece220zjui)
(but do not share your code, test cases or reports). Suggestions are also welcomed. Remember that the tool is only
to **assist** your work. Even if it can't find any issue, it's **not** guaranteed that you will get the full score,
and vice versa.

## Easy Test

{{EASY_TEST_REPORT}}


## Regression Test

{{REGRESSION_REPORT}}

## Report

{{MP3_SUBROUTINE_REPORT}}

_

{{REPORT}}

## How to Use Test Cases (Advanced)

If an issue is detected, a corresponding test case will be generated in the folder `test******`. The test data is in
the asm file. You may copy its content and test your subroutine yourself.

The lcs file is the lc3sim script for you to debug. We have provided a script file for you. Download or checkout this
branch. In current folder, run the command:

```
./replay.sh <index>
```

where `<index>` is the decimal index of the test case, and the script will launch lc3sim for you, where you can debug.
If you can't execute the script, you may need:

```
chmod +x replay.sh
```

Notice that the replay always uses your code of current commit, rather than your latest code.