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
| WMEXE | 47a3 apply_hit_effects | applyHitEffects | 300 | match |
| WMEXE | 4c48 do_attack | doAttack | 300 | fixed bug (kind -1 of a confused fighter's 2nd attack reads AC[7]; Go panicked) |
| WMEXE | 88b1 | weakenGroup | 300 | fixed bug (who >= 0 is a count from a random first member: targetRun) |
| WMEXE | 8bdb | chaosStrike | 300 | fixed bug (same targeting) |
| WMEXE | b03c do_cast_action | doCastAction | 300 | match |
| WMEXE | b44d do_use_item_action | doUseItemAction | 300 | match (inputs: real item records only) |
| WMEXE | 9405 do_cast_spell | doCastSpell | 300 | match |
| WMEXE | c064 do_hide | doHide | 300 | fixed bug (monster actor crashed trySkillGain); open: monster members 6-8, the C's try_raise_skill writes past party |
| WMEXE | c1e4 do_run_away | doRunAway | 300 | match |
| WMEXE | b934 breath | doBreath | 300 | match |
| WMEXE | c3ee do_advance_retreat | doAdvanceRetreat | 300 | match |
| WMEXE | 8fa8 summon | summonGroup | 300 | match |
| WMEXE | c66e perform_action | performAction | 300 | match |
| WMEXE | d109 run_combat_round | runCombatRound | 300 | match |
| WMELE | 724f end_combat_round | endCombatRound | 300 | match (covers update_party_after_round, compact_monster_groups, update_group, tick_combat_effects) |
| WMELE | 574b advance_monster_groups | advanceMonsterGroups | 300 | fixed bug (power sums wrap int16; message fetched twice) |
| WMELE | 62ce start_encounter | setUpEncounter | 300 | fixed bug (g4330/g4326 were separate vars aliasing pic_slot_monster/loaded[4..6]); covers generate_monster_group, check_group_surprise, fill_party_group |
| WMELE | 70ce redraw_combat_screen | redrawCombatScreen | 300 | match (arrange_monster_positions, advance) |

Known limits: a Go write to a field the case neither reads nor expects to change is not seen. UI, sprites, picture/sound loading and end_encounter are stubbed on both sides (MEXE_UI, MELE_UI; dtLog guards in Go).

## Maze (WMAZE, WDOPT, WINIT)

Specs in specs_maze.py, Go in difftest_maze_test.go. `mazeA`/`mazeB` are single-pointer
roots (PTRS count 0; paths `mazeA.Walls0[3]`). The C's show_message_text_m, puzzle_event,
riddle_event, locked_door_menu, load_sound_bank and the drawing/waiting functions are
stubbed; their Go ports return at once under `dtLog`, and keyAvailable is false.

| Overlay | Function | Go | Cases | Status |
|---|---|---|---|---|
| WMAZE | 55b7 tick_effect_timers | tickEffectTimers | 300 | match |
| WMAZE | 61f8 heal_member | healMember | 300 | match |
| WMAZE | 63f4 rest_party | restParty | 300 | match (harness: key_available stub skips step_random) |
| WMAZE | ae49 trap_event | trapEvent (+trapDamageMember, inflictCondition, inflictStatus) | 300 | match (harness: number log via itoa) |
| WMAZE | 5949 move_party | moveParty | 300 | match |
| WMAZE | 56f4 try_move | tryMove | 300 | match |
| WMAZE | 7868 flee_party | fleeParty (+tryStep, canEnterCell) | 300 | match |
| WMAZE | c8eb run_cell_events | runCellEvents (all 25 event kinds) | 300 | match (harness: drawMazeView not stubbed in Go; inputs: event fields in game ranges) |
| WMAZE | a4f5 check_secret_detect | checkSecretDetect | 300 | match |
| WMAZE | ccd2 search_cell | searchCell | 300 | match |
| WMAZE | db1e handle_locked_door | handleLockedDoor | 300 | match |
| WMAZE | 4735 flood_mark_cells | floodMarkCells | 300 | match |
| WMAZE | 9a24 try_skill_gain | trySkillGain | 300 | match |
| WDOPT | 6054 | applySpellOutsideCombat | 300 | match |
| WDOPT | 517b try_open_lock | tryOpenLock | 300 | match |
| WDOPT | 6561 check_item_location_event | wdoptItemLocationEvent | 300 | match |
| WDOPT | 4faf use_item_charge | useItemCharge | 300 | match |
| WDOPT | 4e09 remove_item | removeItem | 300 | match |

Open: force_lock (ced8), pick_lock (d3b3), locked_door_menu (d8a9), puzzle_event, riddle_event,
do_use_item, do_cast_spell: they loop on input, which the harness stubs to nothing (would need
scripted key/click returns). save_game, load_maze_level/alloc_maze_buffers: disk and far buffers.

Skipped (UI, drawing, input, disk, dispatch): WMAZE view and panel drawing (4ac7 4ba8 4e89 7d8c
8175 8332 84f1 8f79 903b 98cb), input/menus (4f2c 5534 5ad8 606f 6bcd 6cf8 7020 9a66 9aae 9c19
9cde 9d0c 9e51), text/notice (53b8 53e5 6631 77ea 7859 a421 aa50 c3e5), door/view animation (b3ac
b500), 9948 sound load; WDOPT ovmain, menus, map and spell-list drawing (4688..48e0 5273 5357..5be2
6737..6bfd 76d5 80d3 8aa9 9531 95f3 9e83); all of WINIT (file loading, credits, title,
graveyard, screen setup).

## Character (WPCMK, WPCVW, WBASE, ROOT helpers)

Specs in specs_char.py, Go in difftest_char_test.go; case names carry the overlay
(`pcmk_`, `pcvw_`, `base_`, `root_`) since addresses repeat across overlays. Overlay BSS
globals go in `OVL_GLOBALS[ovl]` in layout.py (overlays share DS addresses); `bool16` is
a C int flag kept as a Go bool.

| Overlay | Function | Go | Cases | Status |
|---|---|---|---|---|
| WPCMK | 9341 init_new_character (8aed 8786 8e47 8de0 83f0 837b) | initNewCharacter | 300 | match |
| WPCMK | 7274 compute_eligible_classes (7212) | computeEligibleClasses | 300 | match |
| WPCMK | 73e9 apply_class_minimums | checkClassEligible(apply) | 300 | match |
| WPCMK | 7532 apply_race_base_stats (71df) | raiseStatsToMinimum | 300 | match |
| WPCMK | 75d2 apply_sex_modifiers | applySexModifiers | 300 | match |
| WPCMK | 7e92 (7e5f) | initNPCAttitudes | 300 | match |
| WPCMK | 81ad give_class_items (815f 8050) | addStartingItem loop | 300 | match |
| WPCMK | 55ad set_class_skill_masks | setClassSkillMasks | 300 | match |
| WPCMK | 66ea list_realm_spells | listRealmSpells | 300 | match (after harness fix: sentinel) |
| WPCVW | f8c6 check_level_up | checkLevelUp | 300 | match |
| WPCVW | f784 do_level_up | doLevelUp | 300 | match |
| WPCVW | f6e6 / f568 / ed21 / eff8 | updateRank / levelUpStats / levelUpHPStamina / levelUpCombat | 300 each | match (ed21: hp text via itoa for the num log) |
| WPCVW | ea8d roll_skill_points / ea25 raise_skill | rollSkillPoints / raiseSkill | 300 each | match |
| WPCVW | f201 | levelUpSpellPoints | 300 | match |
| WPCVW | 8353 learn_new_spell | learnNewSpell | 300 | match (choose_spell stub: lowest offered spell) |
| WPCVW | 6b3c cast_spell | castSpell | 300 | match |
| WPCVW | 8fbf do_use_item | useItemOnSheet | 300 | match (choose_item, choose_member stubbed: 0) |
| WPCVW | c8c2 can_equip_item | canEquipItem | 300 | match |
| WPCVW | cdb3 (cc36 ccd2 cbf4) | resetCombatValues | 300 | match |
| WPCVW | c726 | sortEquippedFirst | 300 | match |
| WPCVW | 5370 calc_carried_weight (5311) | calcCarriedWeight | 300 | match |
| WPCVW | a1f9 calc_allowed_classes (a1a7) | calcAllowedClasses | 300 | match |
| WPCVW | a11b / a4b1 / a368 | limitStats (by string, race, class) | 300 each | match |
| WPCVW | 76ee count_known_spells | countKnownSpells | 300 | match |
| WPCVW | 5c24 / 5d5b / 5ec7 / 5f53 | clearHandAttack / removeItemShown / removeSwagItem / useItemChargeShown | 300 each | match |
| WPCVW | 9c8e try_combine_recipe (9b3f) | tryCombineRecipe | 300 | match |
| WPCVW | 6aec clamp_toward | clampToward | 300 | match |
| WBASE | 6b30 / 6fd4 | removePartyMember / removeAllMembers | 300 each | match (after harness fix: memberPortrait is int[6]) |
| WBASE | 5190 | findFreePortraitSlot | 300 | match |
| ROOT | 09e2 random / 2556 random_word | random / randomWord | 300 each | match |
| ROOT | 2881 set_bit / 28af test_bit / 28da set_bitfield / 2925 get_bitfield | setBit / testBit / setBitfield / getBitfield | 300 each | match |
| ROOT | 2962 extract_bits | extractBits | 300 | fixed bug (C shifts the unsigned 16-bit word) |

Harness bugs fixed: the fake return address (hlt at CS:FFF0) was DS:0270, i.e.
spell_table[67].realm, now CS:FD70; layout `memberPortrait` was `[6]uint8`, the C has
`int[6]`.

Skipped (UI, input loops, disk, drawing): WPCMK create_character, roll_karma,
distribute_bonus_points, choose_race/sex/class, choose_portrait, choose_starting_spell,
select_spell, skill_points_screen, distribute_skill_points, menus, roster list/rename/delete,
draw_*; WPCVW view_character, do_equip_items, do_swag_menu, do_trade_items,
do_merge_items, do_cast_menu, choose_spell, do_change_class (its mechanics a4b1/a368 are
tested), do_skills_menu, spend_skill_points, do_modify_menu, rename/portrait, get_load_category
(struct return; covered by its callers' drawing only), draw_*; WBASE load_game, add_party_member
(disk), start_game, choose_*, menus, camp scene, game configuration, check_magic_word;
ROOT windows, text, sound, disk, video, keyboard.
