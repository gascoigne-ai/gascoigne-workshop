# Task loop

The loop is the `task-loop` skill (`.harness/skills/task-loop`): run it to work the queue. This file
only adds what is particular to this repo.

- Which task runs next comes from the task files' headers, not from a list here: `status`,
  `depends_on` and `not_before` (the header table is in the common harness README). A task that is
  done is deleted by its commit, which is what unblocks the tasks that depend on it.
- Each task says in its header what it may touch (`scope`) and its commit subject (`commit`).
- The main session checks the results itself; it does not trust an agent's report. Nothing is pushed.
- The check is `make check`. A UI change also has a `make shot` PNG that was looked at, and golden
  files change only where the task changes the look.
- Ambiguity: when a task can be read in more than one way or a choice changes behavior the user
  would notice, stop and ask the user with a recommendation first, record the answer in the task
  file (and, when it outlives the task, in the doc comment of the code it shapes), then continue. Never decide silently. The decisions already made are in
  each task file.
