import { CollectionViewer } from '@angular/cdk/collections';
import { DataSource } from '@angular/cdk/table';
import { catchError, finalize } from 'rxjs/operators';
import { BehaviorSubject, Observable, of } from 'rxjs';
import { LogEntry } from '../models/logEntry';
import { DataService } from './data.service';


export class LogDataSource implements DataSource<LogEntry> {
  private logSubject = new BehaviorSubject<LogEntry[]>([]);
  private loadingSubject = new BehaviorSubject<boolean>(false);
  public loading$ = this.loadingSubject.asObservable();

  constructor(private dataService: DataService) { }
  connect(collectionViewer: CollectionViewer): Observable<LogEntry[]> {
    return this.logSubject.asObservable();
  }
  disconnect(collectionViewer: CollectionViewer): void {
    this.logSubject.complete();
    this.loadingSubject.complete();
  }
  loadLogs(host: string, target: string, sortDirection: string, level: string, limit: number, offset: number) {
    this.loadingSubject.next(true);
    this.dataService.findLogs(host, target, sortDirection, level, limit, offset).pipe(
      catchError(() => of([])),
      finalize(() => this.loadingSubject.next(false))
    ).subscribe(logs => this.logSubject.next(logs));
  }

}
