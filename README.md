# Sway Keyboard Layout and Time

A status program for [swaybar](https://swaywm.org/).
It requires Sway 1.2 for keyboard layout change notifications.

Use it in Sway configuration like this:

    bar {
        status_command sklt
    }

Sway has per-device layouts. This program outputs only the last layout that changed.
When a new device is connected, its initial layout is shown.

## Command Line Options

SKLT accepts command line options that control its output. For the list of options, run:

    sklt -h

### -t *interval*

Time update interval. Valid values for *interval* are (case-insensitive):

* **s** or **second**
* **m** or **minute**
* **h** or **hour**
* **d** or **day**

Default update interval is 1 minute. Maximum interval is 1 hour, but "day" selects date-only format.

### -f *format*

Time format as understood by [Go time package](https://golang.org/pkg/time/).
That is, how the time "Mon Jan 2 15:04:05 -0700 MST 2006" should be formatted.

The default format when updating every second is `2006-01-02 15:04:05` (year-month-day hour:minute:second).
For longer update intervals, smaller units are excluded.

To use a format that contains spaces in the Sway configuration file, either enclose it in escaped quotes (`\"` or `\'`) or escape the spaces (add a backslash before each space):

    status_command sklt -f \"2006-01-02 15:04\"
    status_command sklt -f \'2006-01-02 15:04\'
    status_command sklt -f 2006-01-02\ 15:04
