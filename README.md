# Ambient Glance

Ambient Glance is program powering an ambient information display based on a
LD220-HP POS pole display (normally used in the context of a retail Point Of
Sale).  Ambient Glance features a driver for the display, a TUI mirror used for
development/debugging, individual "apps" that can be shown, and a scheduler
supporting both an automatic carousel between apps and preemption.

The following apps are implemented today:
* Clock
* Fortune - based on the fortune(6) command
* 8 clap - The UCLA 8 clap
* ADS-B - to display the closest detected aircraft within a small radius of a
  receiver
* [OneBusAway](https://onebusaway.org/) - Transit arrival data

> This is not an officially supported Google product. This project is not
> eligible for the [Google Open Source Software Vulnerability Rewards
> Program](https://bughunters.google.com/open-source-security).

![TUI showing AAY282](docs/aay282-full-tui.png)

![LD220-HP pole display showing SKW3853](docs/skw3853.jpg)


## Sample config

```json
{
  "adsb": {
    "tar1090_endpoint":"http://localhost:30152",
    "lat":"",
    "lon":"",
    "radius":"3"
  },
  "oba": {
    "key": "api key goes here",
    "stops": ["stop code"],
    "route_alias": {
      "route code": "NAME"
    }
  }
}
```

The ADS-B app resolves routes from the
[VRS standing data](https://github.com/vradarserver/standing-data) files
mirrored at `https://vrs-standing-data.adsb.lol/`. It downloads them on start
and checks for a newer copy every 6 hours. The files are kept in an
`ambient-glance` directory under the user cache directory. Set
`adsb.route_data_dir` to keep them elsewhere.
