#dump navigation commands
Commands to navigate a Plan 9 style dump

Consists of two programs hist and yest and a package which supports both.
The two commands hist and yest mirror the commands yesterday(1) (http://man.cat-v.org/plan_9/1/yesterday)
and history(1) (http://man.cat-v.org/plan_9/1/history) from plan 9 with some peculiarities.
Both programs expect two enviroment variables, containing paths separated by colons:

	export MAINROOT=/newage/NEWAGE:/bla:/ble
	export DUMPROOT=/dump:/asd:/e/a

The first directoy of each variable which exists will be used. MAINROOT has the root of the
filesystem which is backed up by the dump. DUMPROOT is the root of the dump. Paths in the dump
are of the form

	/dump/yyyy/ddmm/hhmm/rootname/bla/bla

where rootname is obtained from the MAINROOT path. An example:

	/dump/2017/0510/1605/NEWAGE/paurea/

Directly inside the dump root there can be some files, which are ignored by this commands, 
"current", "current_chk", "first" and "lost+found".

YEST(1)

	yest [-y=n] [-m=n] [-d=n] [-h=n] [-D] file_path

The command yest(1) prints the path of the backup file or directory for the path given as
command line.

They -y -m -d -h options give yest the number of years, months days and hours ago
for the dump required. The path printed is the biggest available smaller or equal than the
one requested. By default, yest prints yesterday's file (i.e. yest -d 1).

 The option -D is for debugging the program itself.

HIST(1)

	hist [-Dvc] [-ymdh] file_path

Hist(1) prints the history of a path. by default if it represents a text file, it will print the diffs
of the changes as the file was modified in history. If it is not a text file, or if the -c option is given
it will print the history of creation, deletions and modifications of the file.
 The option -v adds extra information about the changes to the file or directory.

The -y -m -d -h options filters the history, considering one file or less per year, month, day or hour.

 The option -D is for debugging the program itself.

```

    go get github.com/paurea/dump
