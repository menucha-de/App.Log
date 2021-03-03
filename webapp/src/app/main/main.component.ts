import { AfterViewInit, Component, OnDestroy, OnInit, ViewChild, ViewEncapsulation } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatPaginator } from '@angular/material/paginator';

import { MatSort, SortDirection } from '@angular/material/sort';
import { IMqttMessage } from 'ngx-mqtt';
import { BehaviorSubject, merge, Observable, Subject, Subscription } from 'rxjs';
import { tap, takeUntil } from 'rxjs/operators';
import { LogDialogComponent } from '../log-dialog/log-dialog.component';
import { HostStates } from '../models/hostStates';


import { DataService } from '../service/data.service';
import { LogDataSource } from '../service/log.service';
import { LogMqttService } from '../service/logmqtt.service';
import { SettingsComponent } from '../settings/settings.component';
@Component({
  selector: 'app-main',
  templateUrl: './main.component.html',
  styleUrls: ['./main.component.scss']
})
export class MainComponent implements OnInit, AfterViewInit, OnDestroy {
  @ViewChild(MatPaginator) paginator: MatPaginator;
  @ViewChild(MatSort) sort: MatSort;


  dataSource: LogDataSource;
  level = 'ALL';
  target = 'ALL';
  host = 'ALL';
  levels$: Observable<string[]>;
  hosts$ = new BehaviorSubject<Map<string, HostStates>>(new Map());
  private readonly icons = new Map<string, string>([
    ['PANIC', 'whatshot'],
    ['FATAL', 'priority_high'],
    ['ERROR', 'error'],
    ['WARNING', 'warning'],
    ['INFO', 'info'],
    ['DEBUG', 'bug_report'],
    ['TRACE', 'search'],
  ]);

  displayedColumns = ['level', 'date', 'host', 'service', 'thrown'];
  // message = ['message'];
  size: number;
  subscription: Subscription;
  private _onDestroy = new Subject();
  constructor(
    private dataService: DataService,
    private readonly logMqtt: LogMqttService,
    public dialog: MatDialog) { }
  ngOnDestroy(): void {
    this._onDestroy.next();
  }
  ngOnInit(): void {
    this.dataSource = new LogDataSource(this.dataService);
    this.dataSource.loadLogs('ALL', 'ALL', 'desc', 'ALL', 10, 0);
    this.levels$ = this.dataService.getLevels().pipe(tap(levels => {
      if (levels.length > 0) {
        this.level = levels[levels.length - 1];
      }
    }));
    this.dataService.getHosts().pipe(tap(data => {
      const hosts = new Map(Object.entries(data));
      return hosts;
    })).subscribe((data) => this.hosts$.next(data));

    this.subscription = this.logMqtt.topic('log/#').pipe(takeUntil(this._onDestroy)).subscribe((data: IMqttMessage) => {
      const payload = data.payload.toString();
      try {
        const msg = JSON.parse(payload);
        if (this.size < this.paginator.pageSize || this.sort.direction === 'desc' as SortDirection) {
          this.loadLogsPage();
        } else {
          this.size++;
        }
      } catch (error) {
        console.log(error);
      }
    });
    this.subscription = this.logMqtt.topic('topic').pipe(takeUntil(this._onDestroy)).subscribe((data: IMqttMessage) => {
      const payload = data.payload.toString();
      try {
        const msg = JSON.parse(payload);
        const hh = this.hosts$.getValue();

        if (hh[msg.host] !== undefined) {
          if (hh[msg.host].indexOf(msg.target) < 0) {
            const aa = hh[msg.host];
            aa.unshift(msg.target);
            hh[msg.host] = aa;
            this.hosts$.next(hh);
          }
        } else {

          hh[msg.host] = [msg.target, 'ALL'];
          this.hosts$.next(hh);
        }
      } catch (error) {
        console.log(error);
      }
    });
  }

  ngAfterViewInit() {
    this.sort.sortChange.subscribe(() => this.paginator.pageIndex = 0);
    this.dataService.getSize(this.host, this.level, this.target).subscribe((data) => {
      this.size = data;
    });
    merge(this.sort.sortChange, this.paginator.page).pipe(
      tap(() => this.loadLogsPage())).subscribe();

  }
  loadLogsPage() {
    this.dataSource.loadLogs(this.host, this.target, this.sort.direction,
      this.level, this.paginator.pageSize, this.paginator.pageIndex * this.paginator.pageSize);
    this.dataService.getSize(this.host, this.level, this.target).subscribe((data) => {
      this.size = data;
    });
  }
  loadLogsPage1() {
    this.paginator.pageIndex = 0;
    this.loadLogsPage();
  }
  loadTargets() {
    this.target = 'ALL';
    this.paginator.pageIndex = 0;
    this.loadLogsPage();
  }
  getDate(date: number) {
    const d = new Date(date);
    return `${d.getFullYear()}-${this.withZero(d.getMonth() + 1)}-${this.withZero(d.getDate())} ${d.toLocaleTimeString()}.${d.getMilliseconds()}`;
  }
  private withZero(num: number): string {
    return num.toString().padStart(2, '0');
  }
  onRowClicked(row) {
    this.dialog.open(LogDialogComponent, {
      data: row
    });
  }
  onExport() {
    const exportUrl = `rest/log/${this.host}/${this.target}/${this.level}/export`;
    window.open(exportUrl, '_blank');
  }
  onSettings() {
    const dialogref = this.dialog.open(SettingsComponent);
    const sub = dialogref.componentInstance.onDelete.subscribe(() => this.loadLogsPage());
    dialogref.afterClosed().subscribe(() =>
      sub.unsubscribe());
  }

  getIcon(level: string) {
    return this.icons.get(level);
  }

}