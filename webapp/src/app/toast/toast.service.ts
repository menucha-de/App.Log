import { MatSnackBar } from '@angular/material/snack-bar';
import { Injectable } from '@angular/core';
import { ToastComponent } from './toast.component';
@Injectable({
    providedIn: 'root'
})
export class ToastService {
    constructor(private snackBar: MatSnackBar) { }
    public openSnackBar(mess: string, type, duration?) {
        const _snackType = type !== undefined ? type : 'Success';

        this.snackBar.openFromComponent(ToastComponent, {
            duration: duration || 4000,
            panelClass: ['toaster'],
            data: { message: mess, snackType: _snackType, snackBar: this.snackBar }
        });
    }
}
