# Differential tests against the original code

Each tested Go function runs against the original machine code on random cases: same
named input fields, args and RNG seeds; the return value, changed fields, `random` calls
and message/number/sound log must match. `go test -tags nofakecgo -run TestDiff ./internal/game/`
(no Python; it reads `testdata/diff/*.json.gz`, plus SCENARIO.DBS from `$WIZ6_DATA` or
../wiz6-re/gog/x/app for functions that read records).

## How to add a function

1. **Spec** (../wiz6-re/tools/difftest/specs_<area>.py, one file per area; shared bits in
   specs.py): `name = Spec('OVL', 0xADDR, make, stubs=UI_FUNCS, void=...)`.
   `make(rnd)` returns `(state, args[, cargs[, raw]])`: `state` is {Go field path: value}
   (e.g. `'party[2].Skills[3]'`, `'monsterGroups[1].Members[0].HP'`), `args` go to Go,
   `cargs` to the C when they differ (pointers), `raw` {DS offset: bytes} for pointed-to
   buffers (use `SCRATCH`). Stub the overlay's UI functions (draw, animate, delay, input);
   root UI/disk/sound are stubbed already, `random` and pure helpers run.
   If a stub has a lasting effect on mechanics state, add it to `STUB_FX[ovl]`.
2. **Globals**: a new global goes in `GLOBALS` in layout.py (Go name -> DS offset, Go
   type) and in the area's Go roots map. Struct layouts come from the Go types.
3. **Generate**: `.venv/bin/python tools/difftest/gen.py name[,name...]|all [N]` (300 cases
   is enough). It prints unmapped DS writes/reads (missing globals or scratch to ignore:
   `IGNORE_OVL` in gen.py) and discards cases where the original faults.
4. **Go**: add `"name": func(a []int) int { ... }` to the area's map in
   internal/game/difftest_<area>_test.go (its init registers the map and roots).
   Display, delays and sound are off while `dtLog` is set: guard a Go UI function with
   `if dtLog != nil { return }` when the C side stubs it.
5. Every divergence is a port bug (fix Go to match the C) or a harness bug (fix the
   spec/inputs). Note which in the table.

## Combat (WMEXE, WMELE)

| Overlay | Function | Go | Cases | Status |
|---|---|---|---|---|
| WMEXE | f7ddc | damageRoll | 1000 | fixed bug (int16 sum wrap) |
| WMEXE | f7af7 | resistReduce | 1000 | fixed bug (long→int truncation, random(32768)) |
| WMEXE | 7b59 do_damage_effect | damageEffect | 1000 | match |
| WMEXE | 7c5a do_status_effect | statusEffect | 1000 | fixed bug (rollDice int16 wrap) |
| WMEXE | 7848 kill_combatant | killCombatant | 1000 | match |
| WMEXE | f7a16 | effectMsg | 1000 | match |
| WMEXE | f8064 | groupWard | 1000 | match |
| WMEXE | f80cc | wardedDice | 1000 | match |
| WMEXE | f8149 | hitDamage | 1000 | match |
| WMEXE | f825f | hitEffect | 1000 | match |
| WMEXE | f83f3 | hitDamageThen | 1000 | match |
| WMEXE | f86aa | groupStatus | 1000 | match |
| WMEXE | f4572 | fatigueMod | 1000 | match |
| WMEXE | 77d4 try_raise_skill | mexeRaiseSkill/trySkillGain | 1000 | fixed bug (level*factor int16 wrap) |
| WMEXE | 7254 reduce_field6 | reduceStamina | 1000 | fixed bug (int16 wrap) |
| WMEXE | 8861 clamp_toward_x | clampToward | 1000 | fixed bug (int16 wrap) |
| WMEXE | fc4e7 | nextLivingMember | 1000 | match |
| WMEXE | fc546 | nextTargetMonster | 1000 | match |
| WMEXE | 7816 calc_spell_level_value | spellPower | 1000 | match |
| WMEXE | 4c48 do_attack | doAttack | 1000 | match (inputs: thrown attacks need an item; the C indexes item 255 otherwise, Go guards) |
| WMEXE | 47a3 apply_hit_effects | applyHitEffects | 1000 | match |
