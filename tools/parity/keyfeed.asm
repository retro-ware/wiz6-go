; KEYFEED.COM: a TSR that types keys into the BIOS keyboard buffer on emulated time.
; INT 1Ch (18.2 Hz) counts down; at zero the next table entry's key goes in.
; parity.sh appends the table: words (ticks before this key, scan<<8|ascii), ended by 0FFFFh.
	org 100h
start:	jmp init
old1c:	dd 0
kptr:	dw table
cnt:	dw 0
isr:	push ax
	push bx
	push dx
	push si
	push ds
	push cs
	pop ds
	cmp word [cnt], 0
	je fire
	dec word [cnt]
	jmp chain
fire:	mov si, [kptr]
	cmp word [si], 0FFFFh
	je chain
	mov ax, [si+2]
	push ds
	mov bx, 40h
	mov ds, bx
	mov bx, [1Ch]
	mov dx, bx
	add dx, 2
	cmp dx, [82h]
	jb nowrap
	mov dx, [80h]
nowrap:	cmp dx, [1Ah]
	je full			; buffer full: try again next tick
	mov [bx], ax
	mov [1Ch], dx
	pop ds
	add si, 4
	mov [kptr], si
	mov ax, [si]
	mov [cnt], ax
	jmp chain
full:	pop ds
chain:	pop ds
	pop si
	pop dx
	pop bx
	pop ax
	jmp far [cs:old1c]
init:	mov ax, 351Ch
	int 21h
	mov [old1c], bx
	mov [old1c+2], es
	mov ax, [table]
	mov [cnt], ax
	mov ax, 251Ch
	mov dx, isr
	int 21h
	mov dx, (tend - start + 100h + 15) / 16
	mov ax, 3100h
	int 21h
table:
