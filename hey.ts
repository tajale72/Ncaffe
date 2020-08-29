import { Component } from '@angular/core';

interface Information {
  name : string
  lastname : string
}

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
    public name = '';
    public lastname = '';

  public infos: Information[] = [
    {
      name: 'Romit',
      lastname : 'Tajale'
    },
    {
      name: 'first one',
      lastname : 'second one'
    }

  ]

  addpost() {
    this.infos.push({
      name : this.name,
      lastname: this.lastname
    })
  }
  
}


