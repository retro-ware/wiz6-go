# Differential tests against the original code

Each function's Go port is run against the original machine code (Unicorn, in
../wiz6-re/tools/difftest) on 1000 random cases: `go test -tags nofakecgo -run TestDiff ./internal/game/`.
Regenerate cases with `../wiz6-re/.venv/bin/python ../wiz6-re/tools/difftest/gen.py FUNC 1000`.

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
