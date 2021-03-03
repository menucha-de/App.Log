import { Component, EventEmitter, OnInit } from '@angular/core';
import { BehaviorSubject,  from, Observable } from 'rxjs';
import { map, switchMap, tap, toArray } from 'rxjs/operators';
import { DataService } from '../service/data.service';
import { LogTargetConfig } from '../models/log-target-config-model';
import { ToastService } from '../toast/toast.service';
import { HostStates } from '../models/hostStates';
@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent implements OnInit {
  targets: string[];
  logConfigs$ = new BehaviorSubject<LogTargetConfig[]>([]);
  displayedColumns = ['host', 'target', 'level', 'clear'];
  levels$: Observable<string[]>;
  level: string;
  private loadingSubject = new BehaviorSubject<boolean>(false);
  public loading$ = this.loadingSubject.asObservable();
  onDelete = new EventEmitter();

  constructor(private dataService: DataService, private toast: ToastService) { }

  ngOnInit() {
    this.levels$ = this.dataService.getLevels().pipe(tap(levels => {
      if (levels.length > 0) {
        this.level = levels[levels.length - 1];
      }
    }));

    this.dataService.getHosts().pipe(
      map(data => {
        const hosts = new Map(Object.entries(data));
        hosts.delete('ALL');
        const result = new Array<LogTargetConfig>();
        hosts.forEach((value: HostStates, key: string) => {
          for (let i = value.targets.length - 1; i >= 0; i--) {
            if (value.targets[i].name !== 'ALL') {
              result.push(new LogTargetConfig(key, value.targets[i].name, value.targets[i].level));
            }
          }
        });
        return result;
      }))
      .pipe(
        switchMap(targets => from(targets)),

         toArray(),
        tap(targetConfigs => {
          targetConfigs.sort((x, y) => {
            if (x.host.localeCompare(y.host) === 0) {
              return x.target.localeCompare(y.target);
            }
            return x.host.localeCompare(y.host);
          });
          this.logConfigs$.next(targetConfigs);
        })
      ).subscribe();
  }

  setLogLevel(host: string, target: string, level: string) {
    this.dataService.setLogLevel(host, target, level).subscribe(
      () => { }, err => {
        this.toast.openSnackBar(err.error, 'Error');
      });
  }
  deleteLogs(host: string, target: string) {
    this.loadingSubject.next(true);

    this.dataService.deleteLogs(host, target).subscribe(
      () => {
        this.loadingSubject.next(false);
        this.onDelete.emit();
      }, err => {
        this.loadingSubject.next(false);
        this.toast.openSnackBar(err.error, 'Error');
      }

    );
  }
  deleteAll() {
    this.deleteLogs('ALL', 'ALL');
  }
}
