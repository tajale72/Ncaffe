import { User } from './../user';
import { Component, OnInit } from '@angular/core';
import { UserRegistrationService } from '../user-registration.service';

@Component({
  selector: 'app-registration',
  templateUrl: './registration.component.html',
  styleUrls: ['./registration.component.css']
})
export class RegistrationComponent implements OnInit {

  //we will populate the values from HTML form to the 'user'
  //and we will pass same 'user' for RESTAPI call 
  //so for now we will just initialize the object
  user: User = new User("","",0, "");

  message: any;
  constructor(private service:UserRegistrationService) { }

  ngOnInit() {
  }

  public registerNow(){
    let resp=this.service.doRegistration(this.user);
    resp.subscribe((data)=>this.message=data);
      }

}
