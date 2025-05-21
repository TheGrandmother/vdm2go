#!/bin/bash
java -cp '/home/grandmother/.m2/repository/vdmtoolkit/vdm-antlr/1.2.0-SNAPSHOT/vdm-antlr-1.2.0-SNAPSHOT.jar:/home/grandmother/.local/lib/antlr-4.11.1-complete.jar:/home/grandmother/.m2/repository/dk/au/ece/vdmj/vdmj/4.6.0/vdmj-4.6.0.jar' vdmantlr.PrintSTree $@
