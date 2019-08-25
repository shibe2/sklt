# Sway Keyboard Layout and Time

A status program for [swaybar](https://swaywm.org/).
It requires Sway 1.2 for keyboard layout change notifications.

Use it in Sway configuration like this:

    bar {
        status_command sklt
    }

Sway has per-device layouts. This program outputs only the last layout that changed.
When a new device is connected, its initial layout is shown.

For command line reference, run:

    sklt -h
