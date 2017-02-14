POSbot
=========

POSbot is a small Discord bot for monitoring starbases (*aka* POSes) in *EVE Online*, written in Golang.

The bot will constantly retrieve and check the fuel status of a corporation's POSes and alert Discord users via a channel "ping" should the remaining fuel get low.

Installation
------

You can build the application yourself by running the provided commands:
```bash
go get -u github.com/MorpheusXAUT/POSbot
cd $GOPATH/src/github.com/MorpheusXAUT/POSbot
go get ./...
go build
```

If you prefer not to set up Go on your machine, you can use the prebuilt binaries attached to each [release](https://github.com/MorpheusXAUT/POSbot/releases).

Requirements
------

POSbot uses redis to cache starbase data as well as some usage stats, thus requiring you to provide it with a server.

Unfortunately, CCP does not provide detailed location data such as the mapping of `moonIDs` to a location name via any API yet - therefore, POSbot also relies on the single `mapDenormalize` table provided in CCP's [Static Data Export](https://developers.eveonline.com/resource/resources).
You can find a MySQL dump of the required files [here](https://www.fuzzwork.co.uk/dump/), courtesy of [Fuzzwork](https://www.fuzzwork.co.uk). The latest version can usually be retrieved via [this link](https://www.fuzzwork.co.uk/dump/mysql-latest.tar.bz2).
Be aware: the complete SDE will reach about 500MB in size, but you won't need most of the data provided by it. Unfortunately, the `mapDenormalize` table alone is nearly 136MB and thus too large to provide in this repo.

Whilst normal logging all goes to classic `stdout`, POSbot can optionally send its logs to [logz.io](http://logz.io/) or local log files. More details regarding this can be found below.

Usage
------

By default, POSbot will look for its config file - provided in JSON format and by default called `posbot.cfg` - in the folder it's being run from. You can alternatively provide the path to a config file using the `-config=PATH` flag when running the application.

Specifying the `-log=NUMBER` flag, you can modify POSbot's logging behaviour to `stdout` by altering the log level of messages to be displayed. This number can be in the range of 0-5, where 0 will only print panics and 5 will print debug messages as well.
If you find the debug output too spammy, try setting the log level to 4, thus only having POSbot print information about it's operation without additional details.

As the last runtime flag, you can use `-env=ENVIRONMENT` to specify the environment POSbot is being run in. Setting this to `prod` or `production` triggers POSbot's log formatter to output JSON instead of somewhat preformatted text logs, thus allowing for better automatic processing.
It is recommended you set this flag to `prod` if you're not running POSbot manually during development as this setting will also affect the output to logfiles.

Configuration
------

A config-file example without values set is provided as `posbot-empty.cfg`. You can copy this file, rename it to `posbot.cfg` and fill in the values as appropriate. Most values should (hopefully) be somewhat self-explanatory.

### logging
The `logging` section allows you to enable file and/or logz.io based logging. You can specify separate files for each log level or have them all go to a single file by putting the same path as every value.
Note that POSbot will use relative paths starting at its working directory unless you specify an absolute path for the logfiles. The user POSbot is being run from needs read/write access to the file location you've specified.

The `context` object in the logz.io config section allows you to provide additional information that will be added to every log entry sent to logz.io, thus allowing you to further distinguish different applications.

### discord
To authenticate POSbot with Discord, you will have to [register a Discord application](https://discordapp.com/developers/applications/me/create). Choose whatever name you want as an app name, however POSbot will currently refer to itself as "POSbot", thus making this the recommended value.
Since no users authenticate against POSbot, you don't need to specify any redirect URIs. You can also upload an avatar for POSbot via this form.

Once you've completed the registration of your new Discord app, you can copy the required access token by clicking `click to reveal`. As the last step required for setting up POSbot's Discord elements, you'll need to invite the bot to your server and grant it permissions.
I've found it easiest to use the community-provided [Discord Permissions Calculator](https://discordapi.com/permissions.html). Select the permissions you want POSbot to have and enter your `client ID` (to be found at the app details page you've just created). The site will generate a link for your to click, allowing you to directly invite POSbot to your Discord server.

As of now, POSbot each instance of POSbot can only log to a single Discord server and channel, although this might be extended in the future to allow multiple destinations.
You'll have to specify the target server (called `guild` in Discord) and channel in the `discord` config section. The easiest way to do so it by enabling developer mode in Discord's `appearance` settings, then right-clicking the server as well as channel and selecting `copy ID`.
Furthermore, you can provide a `botAdminRoldID`, allowing for users of said group to execute extended bot commands (currently only displaying stats about the bot's runtime).

After the bot has joined your server (even if it's offline), you can grant it the appropriate permissions to read and post to the channel you want it to.
In its current state, POSbot requires `Read Messages`, `Send Messages`, `Read Message History` and `Mention Everyone` to function properly.

Via using the `notifications` settings for `warning` and `critical`, you can specify the time to way (in seconds) between each notification regarding a POS with the respective fuel status being sent. POSbot repeats its notifications unless the fuel quantity rises above the specified thresholds again.

You can leave the `debug` and `verbose` flags set to `false`, those were mostly used in development.

### eve

Since POS data unfortunately is only available via CCP's old XML API, you'll need to create an EVE [API Key](https://community.eveonline.com/support/api-key/).
The key needs to be created as a directory or CEO of a corporation since it requires the `StarbaseList` and `StarbaseDetails` information as found in the `Outposts and Starbases` category of a corp API key. Copy & paste the API Key ID and vCode into the appropriate config sections.

All access to CCP's new ESI API is using unauthorized endpoints, thus not requiring you to create a separate application there.

Should your corp own multiple starbases, but you only want a certain subset to be monitored, you can exclude some of them using the `ignoredStarbases` array. Simply specify the `starbaseID` of each structure you want to skip, provided as an integer, one per line.

The `monitorInterval` specifies the interval (in seconds) between each fuel check POSbot performs. Whilst checking at a higher interval makes sure you get notifications as early as possible, you don't actually receive a more detailed fuel status since EVE's API only updates these values once per hour (and POS fuel is consumed on an hourly basis as well).
It is thus recommended to keep this value at 5 minutes (*aka* 300 seconds) since this makes sure all information is accurate and updates within a short while after EVE caches expire.

Lastly, the `fuelThreshold` section can be used to modify POSbot's behaviour regarding the fuel status of a POS (both values are in **hours**): once the remaining fuel falls below the `warning` threshold, POSbot will send out a notification using Discord's `@here` mention system, notifying all currently online pilots.
As the fuel approaches the `critical` value, POSbot will resort to more aggressive pinging, mentioning everyone in the channel, thus also pinging offline users. The pings will be repeated after the timespan specified in the `discord > notifications` section.

### redis

The `redis` config section is used to inform POSbot about the location and possible authentication required to connect to the redis server. `address` should be in the form of `HOST:PORT`, `database` allows you to specify the number of a redis DB to choose (default is 0).
Leaving the `password` as an empty string will cause POSbot not to perform any `AUTH` commands upon connection.

### mysql

Same as with the `redis` section, the `mysql` config is used to specify the MySQL server to connect to (containing the `mapDenormalize` table from EVE's SDE). The user provided to POSbot only requires `SELECT` privileges on the `mapDenormalize` table.
Once again, the `address` should be in the form of `HOST:PORT`.

Attribution
------

### Fuzzwork
Thank you [Steve Ronuken](https://evewho.com/pilot/Steve+Ronuken) for providing the EVE Static Data Export in MySQL format at [fuzzwork](https://www.fuzzwork.co.uk/dump/).

### CCP
EVE Online and the EVE logo are the registered trademarks of CCP hf. All rights are reserved worldwide. All other trademarks are the property of their respective owners. EVE Online, the EVE logo, EVE and all associated logos and designs are the intellectual property of CCP hf. All artwork, screenshots, characters, vehicles, storylines, world facts or other recognizable features of the intellectual property relating to these trademarks are likewise the intellectual property of CCP hf. CCP hf. has granted permission to POSbot to use EVE Online and all associated logos and designs for promotional and information purposes on its website but does not endorse, and is not in any way affiliated with, POSbot. CCP is in no way responsible for the content on or functioning of this website, nor can it be liable for any damage arising from the use of this website.

License
------

[MIT License](https://opensource.org/licenses/mit-license.php)