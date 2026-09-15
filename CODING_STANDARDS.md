All production Go code must report no violations in `make messgo`, no baselining, no grandfathering.
All production Go code must report a covered-MSI of 80% or above in `make mutago`, no baselining, no grandfathering.

`make messgo` runs `messgo . text quality-gates/messgo-ruleset.xml --ignore-tests`. The ruleset excludes `ExitExpression` (in `design`) as the single intentional exception: `main()` must call `os.Exit` to propagate the CLI exit code to the shell. 
